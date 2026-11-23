-- 上游余额监控：accounts 表新增上游余额、同步时间与专属预警阈值列。
-- 余额来自上游转售站的 GET {base_url}/v1/usage 接口（balance 字段，美元），
-- 由上游余额同步服务定期刷新；NULL 表示尚未成功同步或上游不支持该接口。

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS upstream_balance NUMERIC(20,8);

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS upstream_balance_updated_at TIMESTAMPTZ;

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS balance_alert_threshold NUMERIC(20,4);

COMMENT ON COLUMN accounts.upstream_balance IS
    'Upstream reseller account balance (USD) fetched from GET {base_url}/v1/usage; NULL until first successful sync or unsupported upstream';

COMMENT ON COLUMN accounts.upstream_balance_updated_at IS
    'Last successful upstream balance sync time';

COMMENT ON COLUMN accounts.balance_alert_threshold IS
    'Per-account upstream balance alert threshold (USD); NULL falls back to the global default in settings';
