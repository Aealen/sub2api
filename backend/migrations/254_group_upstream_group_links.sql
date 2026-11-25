-- 本站分组 ↔ 上游分组的显式关联（多对多）。
--
-- 背景：本站分组与上游分组之间此前没有直接建模，只能靠
-- groups ← account_groups → accounts.upstream_id → upstream_groups 隐式推导，
-- 无法表达「同一个本站分组由某上游的哪个套餐供货」，也无法支持一个分组对应多个上游分组。
-- 本表把这条供货关系变成一等实体，供上游总览页展示与维护。
--
-- 一、上游分组天然归属唯一上游（upstream_groups.upstream_id），因此通过本表即可得到所属上游。
-- 二、唯一约束只限定 (group_id, upstream_group_id)，不限制同一上游的多个分组关联同一本站分组，
--     以便表达「同上游不同套餐分别供货同一分组」。

CREATE TABLE IF NOT EXISTS group_upstream_group_links (
    id                BIGSERIAL PRIMARY KEY,
    group_id          BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    upstream_group_id BIGINT NOT NULL REFERENCES upstream_groups(id) ON DELETE CASCADE,
    -- 该关联的备注（如「主力通道」「备用」），供总览页展示
    note              TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ
);

-- 同一本站分组与同一上游分组之间只保留一条有效关联
CREATE UNIQUE INDEX IF NOT EXISTS idx_group_upstream_group_links_unique_active
    ON group_upstream_group_links (group_id, upstream_group_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_group_upstream_group_links_group
    ON group_upstream_group_links (group_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_group_upstream_group_links_upstream_group
    ON group_upstream_group_links (upstream_group_id)
    WHERE deleted_at IS NULL;

COMMENT ON TABLE group_upstream_group_links IS
    'Explicit many-to-many link between local pricing groups and upstream-side groups (supply relationship)';
COMMENT ON COLUMN group_upstream_group_links.group_id IS
    'Local pricing group supplied by the upstream group';
COMMENT ON COLUMN group_upstream_group_links.upstream_group_id IS
    'Upstream-side group/plan; its upstream_id identifies the vendor';
COMMENT ON COLUMN group_upstream_group_links.note IS
    'Free-text role of this supply path (primary, backup, etc.)';
