-- 上游管理：把隐含在账号 credentials.base_url 中的上游归一为可维护实体，
-- 并建立上游侧分组与上游模型价格台账，用于余额汇总、账号归属与跨上游比价。
--
-- 价格单位统一为 USD per-token（与 billing 目录价、channel_model_pricing 一致），
-- 前端按每百万 token 展示；避免与现有计费链出现量级差异。

-- 1. 上游库
CREATE TABLE IF NOT EXISTS upstreams (
    id         BIGSERIAL PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    -- 归一化后的上游基地址（scheme://host），用于账号归并提示与后台跳转
    base_url   TEXT,
    website    TEXT,
    contact    TEXT,
    note       TEXT,
    status     VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

ALTER TABLE upstreams DROP CONSTRAINT IF EXISTS upstreams_status_check;
ALTER TABLE upstreams ADD CONSTRAINT upstreams_status_check
    CHECK (status IN ('active', 'inactive'));

-- 名称不区分大小写唯一（软删后允许复用）
CREATE UNIQUE INDEX IF NOT EXISTS idx_upstreams_name_active
    ON upstreams (lower(name)) WHERE deleted_at IS NULL;

-- 同一 base_url 只能挂一个上游，避免归并时出现歧义
CREATE UNIQUE INDEX IF NOT EXISTS idx_upstreams_base_url_active
    ON upstreams (base_url)
    WHERE deleted_at IS NULL AND base_url IS NOT NULL AND base_url <> '';

COMMENT ON TABLE upstreams IS 'Upstream vendors (resellers) maintained by admins';
COMMENT ON COLUMN upstreams.base_url IS 'Normalized upstream base URL (scheme://host) used to suggest account assignment';
COMMENT ON COLUMN upstreams.status IS 'active | inactive; inactive upstreams are hidden from comparison by default';

-- 2. 账号归属上游：可空，删除上游时置空，账号保留
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS upstream_id BIGINT REFERENCES upstreams(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_accounts_upstream_id
    ON accounts (upstream_id) WHERE deleted_at IS NULL;

COMMENT ON COLUMN accounts.upstream_id IS
    'Owning upstream vendor; NULL means not yet assigned (assigned via admin upstream merge flow)';

-- 3. 上游侧分组（转售站内部的分组/套餐及其倍率）
CREATE TABLE IF NOT EXISTS upstream_groups (
    id                   BIGSERIAL PRIMARY KEY,
    upstream_id          BIGINT NOT NULL REFERENCES upstreams(id) ON DELETE CASCADE,
    name                 VARCHAR(100) NOT NULL,
    rate_multiplier      NUMERIC(10,4),
    peak_rate_multiplier NUMERIC(10,4),
    peak_start           VARCHAR(8),
    peak_end             VARCHAR(8),
    source               VARCHAR(20) NOT NULL DEFAULT 'manual',
    note                 TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ
);

ALTER TABLE upstream_groups DROP CONSTRAINT IF EXISTS upstream_groups_source_check;
ALTER TABLE upstream_groups ADD CONSTRAINT upstream_groups_source_check
    CHECK (source IN ('manual', 'probe', 'import', 'crawler'));

CREATE UNIQUE INDEX IF NOT EXISTS idx_upstream_groups_name_active
    ON upstream_groups (upstream_id, lower(name)) WHERE deleted_at IS NULL;

COMMENT ON TABLE upstream_groups IS 'Upstream-side groups/plans with their declared rate multiplier';
COMMENT ON COLUMN upstream_groups.source IS 'manual | probe (from /v1/sub2api/billing) | import | crawler';

-- 4. 上游模型价格台账（手工录入 / 批量导入 / 爬虫推送）
CREATE TABLE IF NOT EXISTS upstream_model_prices (
    id                         BIGSERIAL PRIMARY KEY,
    upstream_id                BIGINT NOT NULL REFERENCES upstreams(id) ON DELETE CASCADE,
    upstream_group_id          BIGINT REFERENCES upstream_groups(id) ON DELETE SET NULL,
    -- 上游原始模型名（用于核对上游报价单）
    upstream_model             VARCHAR(200) NOT NULL,
    -- 对齐用的平台标准模型名（跨上游比价的行键）
    model_name                 VARCHAR(200) NOT NULL,
    platform                   VARCHAR(50) NOT NULL DEFAULT '',
    billing_mode               VARCHAR(20) NOT NULL DEFAULT 'token',
    currency                   VARCHAR(8) NOT NULL DEFAULT 'USD',
    input_price_per_token      NUMERIC(20,10),
    output_price_per_token     NUMERIC(20,10),
    cache_read_price_per_token NUMERIC(20,10),
    cache_write_price_per_token NUMERIC(20,10),
    per_request_price          NUMERIC(20,10),
    -- 该报价对应的上游倍率；NULL 表示按上游分组/账号倍率推导
    rate_multiplier            NUMERIC(10,4),
    source                     VARCHAR(20) NOT NULL DEFAULT 'manual',
    source_url                 TEXT,
    effective_at               TIMESTAMPTZ,
    note                       TEXT,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at                 TIMESTAMPTZ
);

ALTER TABLE upstream_model_prices DROP CONSTRAINT IF EXISTS upstream_model_prices_source_check;
ALTER TABLE upstream_model_prices ADD CONSTRAINT upstream_model_prices_source_check
    CHECK (source IN ('manual', 'import', 'crawler'));

ALTER TABLE upstream_model_prices DROP CONSTRAINT IF EXISTS upstream_model_prices_billing_mode_check;
ALTER TABLE upstream_model_prices ADD CONSTRAINT upstream_model_prices_billing_mode_check
    CHECK (billing_mode IN ('token', 'per_request', 'image', 'video'));

-- 同一上游、同一上游分组下同一上游模型只保留一条有效台账
CREATE UNIQUE INDEX IF NOT EXISTS idx_upstream_model_prices_unique_active
    ON upstream_model_prices (upstream_id, COALESCE(upstream_group_id, 0), upstream_model)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_upstream_model_prices_model_name
    ON upstream_model_prices (model_name) WHERE deleted_at IS NULL;

COMMENT ON TABLE upstream_model_prices IS
    'Upstream model price ledger (USD per token) from manual entry, bulk import or crawler push';
COMMENT ON COLUMN upstream_model_prices.model_name IS
    'Platform standard model name used as the row key for cross-upstream comparison';
COMMENT ON COLUMN upstream_model_prices.rate_multiplier IS
    'Rate multiplier attached to this quote; NULL falls back to upstream group/account multiplier';
