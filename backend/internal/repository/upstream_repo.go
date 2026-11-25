package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// upstreamRepository 上游管理数据访问（原生 SQL）。
//
// 表结构见 migrations/253_upstream_management.sql；上游只承载商务元信息，
// 余额/倍率/模型映射仍以 accounts 为事实来源，这里只做聚合查询。
type upstreamRepository struct {
	db *sql.DB
}

// NewUpstreamRepository 创建上游管理仓储。
func NewUpstreamRepository(db *sql.DB) service.UpstreamRepository {
	return &upstreamRepository{db: db}
}

const upstreamColumns = `id, name, COALESCE(base_url, ''), COALESCE(website, ''),
	COALESCE(contact, ''), COALESCE(note, ''), status, created_at, updated_at`

func scanUpstream(scanner interface{ Scan(...any) error }) (*service.Upstream, error) {
	u := &service.Upstream{}
	if err := scanner.Scan(
		&u.ID, &u.Name, &u.BaseURL, &u.Website, &u.Contact, &u.Note, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return u, nil
}

// ---------- upstreams ----------

// CreateUpstream 新建上游。
func (r *upstreamRepository) CreateUpstream(ctx context.Context, u *service.Upstream) error {
	query := `INSERT INTO upstreams (name, base_url, website, contact, note, status)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`
	if err := r.db.QueryRowContext(ctx, query,
		u.Name, u.BaseURL, u.Website, u.Contact, u.Note, u.Status,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return service.ErrUpstreamExists
		}
		return fmt.Errorf("create upstream: %w", err)
	}
	return nil
}

// GetUpstreamByID 按 ID 查询未软删的上游。
func (r *upstreamRepository) GetUpstreamByID(ctx context.Context, id int64) (*service.Upstream, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+upstreamColumns+` FROM upstreams WHERE id = $1 AND deleted_at IS NULL`, id)
	u, err := scanUpstream(row)
	if err == sql.ErrNoRows {
		return nil, service.ErrUpstreamNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get upstream: %w", err)
	}
	return u, nil
}

// GetUpstreamByName 按名称查询（不区分大小写）。
func (r *upstreamRepository) GetUpstreamByName(ctx context.Context, name string) (*service.Upstream, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+upstreamColumns+` FROM upstreams
		 WHERE lower(name) = lower($1) AND deleted_at IS NULL
		 ORDER BY id LIMIT 1`, strings.TrimSpace(name))
	u, err := scanUpstream(row)
	if err == sql.ErrNoRows {
		return nil, service.ErrUpstreamNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get upstream by name: %w", err)
	}
	return u, nil
}

// GetUpstreamByBaseURL 按归一化 base_url 查询。
func (r *upstreamRepository) GetUpstreamByBaseURL(ctx context.Context, baseURL string) (*service.Upstream, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+upstreamColumns+` FROM upstreams
		 WHERE base_url = $1 AND deleted_at IS NULL
		 ORDER BY id LIMIT 1`, strings.TrimSpace(baseURL))
	u, err := scanUpstream(row)
	if err == sql.ErrNoRows {
		return nil, service.ErrUpstreamNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get upstream by base_url: %w", err)
	}
	return u, nil
}

// UpdateUpstream 更新上游元信息。
func (r *upstreamRepository) UpdateUpstream(ctx context.Context, u *service.Upstream) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE upstreams
		 SET name = $2, base_url = NULLIF($3, ''), website = $4, contact = $5, note = $6,
		     status = $7, updated_at = now()
		 WHERE id = $1 AND deleted_at IS NULL`,
		u.ID, u.Name, u.BaseURL, u.Website, u.Contact, u.Note, u.Status)
	if err != nil {
		if isUniqueViolation(err) {
			return service.ErrUpstreamExists
		}
		return fmt.Errorf("update upstream: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrUpstreamNotFound
	}
	return nil
}

// DeleteUpstream 软删上游；账号 upstream_id 由外键 ON DELETE SET NULL，
// 但软删不触发外键，这里显式清空账号归属。
func (r *upstreamRepository) DeleteUpstream(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete upstream: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`UPDATE accounts SET upstream_id = NULL, updated_at = now()
		 WHERE upstream_id = $1 AND deleted_at IS NULL`, id); err != nil {
		return fmt.Errorf("clear account upstream: %w", err)
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE upstreams SET deleted_at = now(), updated_at = now()
		 WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("delete upstream: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrUpstreamNotFound
	}
	return tx.Commit()
}

// ListUpstreams 列出全部未软删上游。
func (r *upstreamRepository) ListUpstreams(ctx context.Context) ([]service.Upstream, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+upstreamColumns+` FROM upstreams WHERE deleted_at IS NULL ORDER BY lower(name)`)
	if err != nil {
		return nil, fmt.Errorf("list upstreams: %w", err)
	}
	defer rows.Close()

	list := make([]service.Upstream, 0, 8)
	for rows.Next() {
		u, scanErr := scanUpstream(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan upstream: %w", scanErr)
		}
		list = append(list, *u)
	}
	return list, rows.Err()
}

// ListUpstreamOverview 上游列表聚合：账号数、余额合计、倍率区间、台账与分组规模。
func (r *upstreamRepository) ListUpstreamOverview(ctx context.Context) ([]service.UpstreamOverview, error) {
	query := `
		SELECT u.id, u.name, COALESCE(u.base_url, ''), COALESCE(u.website, ''),
		       COALESCE(u.contact, ''), COALESCE(u.note, ''), u.status, u.created_at, u.updated_at,
		       COUNT(a.id) FILTER (WHERE a.deleted_at IS NULL) AS account_total,
		       COUNT(a.id) FILTER (WHERE a.deleted_at IS NULL AND a.status = 'active') AS account_active,
		       -- 整站余额语义：同站所有 API Key 查到的 balance 相同（站级余额池），
		       -- 取任一非空值即可（MIN 天然忽略 NULL，同值时无歧义）；
		       -- SUM/MAX 均不妥：SUM 按账号数重复放大，MAX 在多账户时偏向最大值。
		       COALESCE(MIN(a.upstream_balance) FILTER (WHERE a.deleted_at IS NULL), 0) AS balance_total,
		       MAX(a.upstream_balance_updated_at) FILTER (WHERE a.deleted_at IS NULL) AS balance_updated_at,
		       MIN(a.rate_multiplier) FILTER (WHERE a.deleted_at IS NULL) AS rate_min,
		       MAX(a.rate_multiplier) FILTER (WHERE a.deleted_at IS NULL) AS rate_max,
		       (SELECT COUNT(*) FROM upstream_model_prices p
		         WHERE p.upstream_id = u.id AND p.deleted_at IS NULL) AS price_count,
		       (SELECT COUNT(*) FROM upstream_groups g
		         WHERE g.upstream_id = u.id AND g.deleted_at IS NULL) AS group_count
		  FROM upstreams u
		  LEFT JOIN accounts a ON a.upstream_id = u.id
		 WHERE u.deleted_at IS NULL
		 GROUP BY u.id
		 ORDER BY lower(u.name)`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list upstream overview: %w", err)
	}
	defer rows.Close()

	list := make([]service.UpstreamOverview, 0, 8)
	for rows.Next() {
		var item service.UpstreamOverview
		if err := rows.Scan(
			&item.ID, &item.Name, &item.BaseURL, &item.Website, &item.Contact, &item.Note,
			&item.Status, &item.CreatedAt, &item.UpdatedAt,
			&item.AccountTotal, &item.AccountActive, &item.SiteBalance, &item.BalanceUpdatedAt,
			&item.RateMultiplierMin, &item.RateMultiplierMax,
			&item.ModelPriceCount, &item.GroupCount,
		); err != nil {
			return nil, fmt.Errorf("scan upstream overview: %w", err)
		}
		item.GroupNames = []string{}
		list = append(list, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return list, nil
	}

	// 分组名单独查询后回填，避免在聚合 SQL 里做 string_agg 解析
	nameRows, err := r.db.QueryContext(ctx,
		`SELECT upstream_id, name FROM upstream_groups
		 WHERE deleted_at IS NULL ORDER BY lower(name)`)
	if err != nil {
		return nil, fmt.Errorf("list upstream group names: %w", err)
	}
	defer nameRows.Close()
	byID := make(map[int64][]string, len(list))
	for nameRows.Next() {
		var upstreamID int64
		var name string
		if err := nameRows.Scan(&upstreamID, &name); err != nil {
			return nil, fmt.Errorf("scan upstream group name: %w", err)
		}
		byID[upstreamID] = append(byID[upstreamID], name)
	}
	if err := nameRows.Err(); err != nil {
		return nil, err
	}
	for i := range list {
		if names := byID[list[i].ID]; len(names) > 0 {
			list[i].GroupNames = names
		}
	}
	return list, nil
}

// ---------- accounts ----------

// ListUnassignedAccountHosts 未归属上游的 apikey 账号按归一化 host 聚合。
func (r *upstreamRepository) ListUnassignedAccountHosts(ctx context.Context) ([]service.UnassignedAccountHost, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, COALESCE(credentials->>'base_url', '') AS base_url, name
		   FROM accounts
		  WHERE deleted_at IS NULL AND upstream_id IS NULL AND type = 'apikey'
		  ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list unassigned accounts: %w", err)
	}
	defer rows.Close()

	type bucket struct {
		raw   string
		names []string
		ids   []int64
		count int
	}
	buckets := make(map[string]*bucket)
	order := make([]string, 0, 8)
	for rows.Next() {
		var (
			accountID  int64
			rawBaseURL string
			name       string
		)
		if err := rows.Scan(&accountID, &rawBaseURL, &name); err != nil {
			return nil, fmt.Errorf("scan unassigned account: %w", err)
		}
		host := service.NormalizeUpstreamBaseURL(rawBaseURL)
		if host == "" {
			host = strings.TrimSpace(rawBaseURL)
		}
		b, ok := buckets[host]
		if !ok {
			b = &bucket{raw: rawBaseURL}
			buckets[host] = b
			order = append(order, host)
		}
		b.count++
		b.ids = append(b.ids, accountID)
		if len(b.names) < 5 {
			b.names = append(b.names, name)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]service.UnassignedAccountHost, 0, len(buckets))
	for _, host := range order {
		b := buckets[host]
		result = append(result, service.UnassignedAccountHost{
			BaseURL:      b.raw,
			Host:         host,
			AccountCount: b.count,
			SampleNames:  b.names,
			AccountIDs:   b.ids,
		})
	}
	return result, nil
}

// AssignAccountsToUpstream 批量把账号挂到上游（覆盖原归属）。
func (r *upstreamRepository) AssignAccountsToUpstream(ctx context.Context, accountIDs []int64, upstreamID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE accounts SET upstream_id = $1, updated_at = now()
		  WHERE id = ANY($2) AND deleted_at IS NULL`,
		upstreamID, pq.Array(accountIDs))
	if err != nil {
		return fmt.Errorf("assign accounts to upstream: %w", err)
	}
	return nil
}

// ListUpstreamAccounts 列出某上游下的账号（含模型映射与探测到的倍率）。
func (r *upstreamRepository) ListUpstreamAccounts(ctx context.Context, upstreamID int64) ([]service.UpstreamAccountRow, error) {
	rows, err := r.db.QueryContext(ctx, upstreamAccountQuery+` AND a.upstream_id = $1 ORDER BY a.name`, upstreamID)
	if err != nil {
		return nil, fmt.Errorf("list upstream accounts: %w", err)
	}
	defer rows.Close()
	return r.scanUpstreamAccounts(ctx, rows)
}

// ListAllAssignedAccounts 列出全部已归属上游的账号（比价推导用）。
func (r *upstreamRepository) ListAllAssignedAccounts(ctx context.Context) ([]service.UpstreamAccountRow, error) {
	rows, err := r.db.QueryContext(ctx, upstreamAccountQuery+` AND a.upstream_id IS NOT NULL ORDER BY a.name`)
	if err != nil {
		return nil, fmt.Errorf("list assigned accounts: %w", err)
	}
	defer rows.Close()
	return r.scanUpstreamAccounts(ctx, rows)
}

const upstreamAccountQuery = `
	SELECT a.id, a.upstream_id, a.name, a.platform, a.type, a.status,
	       COALESCE(a.credentials->>'base_url', '') AS base_url,
	       a.rate_multiplier, a.upstream_balance, a.upstream_balance_updated_at,
	       COALESCE(a.credentials->'model_mapping', '{}'::jsonb) AS model_mapping,
	       COALESCE(a.extra->'upstream_billing_probe'->'data', '{}'::jsonb) AS probe_data
	  FROM accounts a
	 WHERE a.deleted_at IS NULL`

func (r *upstreamRepository) scanUpstreamAccounts(ctx context.Context, rows *sql.Rows) ([]service.UpstreamAccountRow, error) {
	list := make([]service.UpstreamAccountRow, 0, 8)
	accountIDs := make([]int64, 0, 8)
	for rows.Next() {
		var (
			item         service.UpstreamAccountRow
			modelMapping []byte
			probeData    []byte
		)
		if err := rows.Scan(
			&item.ID, &item.UpstreamID, &item.Name, &item.Platform, &item.Type, &item.Status,
			&item.BaseURL, &item.RateMultiplier, &item.UpstreamBalance, &item.UpstreamBalanceUpdatedAt,
			&modelMapping, &probeData,
		); err != nil {
			return nil, fmt.Errorf("scan upstream account: %w", err)
		}
		item.ModelMapping = decodeStringMap(modelMapping)
		item.MappedModelCount = len(item.ModelMapping)
		applyProbeSnapshot(&item, probeData)
		item.GroupIDs = []int64{}
		item.GroupNames = []string{}
		list = append(list, item)
		accountIDs = append(accountIDs, item.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(accountIDs) == 0 {
		return list, nil
	}

	// 账号所属平台分组
	groupRows, err := r.db.QueryContext(ctx,
		`SELECT ag.account_id, g.id, g.name
		   FROM account_groups ag
		   JOIN groups g ON g.id = ag.group_id
		  WHERE ag.account_id = ANY($1) AND g.deleted_at IS NULL
		  ORDER BY g.name`,
		pq.Array(accountIDs))
	if err != nil {
		return nil, fmt.Errorf("list account groups: %w", err)
	}
	defer groupRows.Close()
	index := make(map[int64]int, len(list))
	for i := range list {
		index[list[i].ID] = i
	}
	for groupRows.Next() {
		var (
			accountID int64
			groupID   int64
			name      string
		)
		if err := groupRows.Scan(&accountID, &groupID, &name); err != nil {
			return nil, fmt.Errorf("scan account group: %w", err)
		}
		if i, ok := index[accountID]; ok {
			list[i].GroupIDs = append(list[i].GroupIDs, groupID)
			list[i].GroupNames = append(list[i].GroupNames, name)
		}
	}
	return list, groupRows.Err()
}

// applyProbeSnapshot 从 accounts.extra.upstream_billing_probe.data 提取上游声明倍率。
func applyProbeSnapshot(item *service.UpstreamAccountRow, raw []byte) {
	if len(raw) == 0 {
		return
	}
	var data struct {
		GroupRateMultiplier     *float64 `json:"group_rate_multiplier"`
		EffectiveRateMultiplier *float64 `json:"effective_rate_multiplier"`
		PeakRateMultiplier      *float64 `json:"peak_rate_multiplier"`
		PeakStart               string   `json:"peak_start"`
		PeakEnd                 string   `json:"peak_end"`
		ObservedAt              string   `json:"observed_at"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return
	}
	item.ProbedGroupRateMultiplier = data.GroupRateMultiplier
	item.ProbedEffectiveRateMultiplier = data.EffectiveRateMultiplier
	item.ProbedPeakMultiplier = data.PeakRateMultiplier
	item.ProbedPeakStart = data.PeakStart
	item.ProbedPeakEnd = data.PeakEnd
	item.ProbedObservedAt = data.ObservedAt
}

func decodeStringMap(raw []byte) map[string]string {
	result := make(map[string]string)
	if len(raw) == 0 {
		return result
	}
	var anyMap map[string]any
	if err := json.Unmarshal(raw, &anyMap); err != nil {
		return result
	}
	for k, v := range anyMap {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}
	return result
}

// ---------- upstream groups ----------

// ListUpstreamGroups 列出某上游的分组。
func (r *upstreamRepository) ListUpstreamGroups(ctx context.Context, upstreamID int64) ([]service.UpstreamGroup, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, upstream_id, name, rate_multiplier, peak_rate_multiplier,
		        COALESCE(peak_start, ''), COALESCE(peak_end, ''), source, COALESCE(note, ''),
		        created_at, updated_at
		   FROM upstream_groups
		  WHERE upstream_id = $1 AND deleted_at IS NULL
		  ORDER BY lower(name)`, upstreamID)
	if err != nil {
		return nil, fmt.Errorf("list upstream groups: %w", err)
	}
	defer rows.Close()

	list := make([]service.UpstreamGroup, 0, 4)
	for rows.Next() {
		var g service.UpstreamGroup
		if err := rows.Scan(
			&g.ID, &g.UpstreamID, &g.Name, &g.RateMultiplier, &g.PeakRateMultiplier,
			&g.PeakStart, &g.PeakEnd, &g.Source, &g.Note, &g.CreatedAt, &g.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan upstream group: %w", err)
		}
		list = append(list, g)
	}
	return list, rows.Err()
}

// GetUpstreamGroup 按 ID 查询分组。
func (r *upstreamRepository) GetUpstreamGroup(ctx context.Context, id int64) (*service.UpstreamGroup, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, upstream_id, name, rate_multiplier, peak_rate_multiplier,
		        COALESCE(peak_start, ''), COALESCE(peak_end, ''), source, COALESCE(note, ''),
		        created_at, updated_at
		   FROM upstream_groups WHERE id = $1 AND deleted_at IS NULL`, id)
	var g service.UpstreamGroup
	err := row.Scan(
		&g.ID, &g.UpstreamID, &g.Name, &g.RateMultiplier, &g.PeakRateMultiplier,
		&g.PeakStart, &g.PeakEnd, &g.Source, &g.Note, &g.CreatedAt, &g.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, service.ErrUpstreamGroupNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get upstream group: %w", err)
	}
	return &g, nil
}

// UpsertUpstreamGroup 按 ID 或 (upstream_id, name) 幂等写入分组。
func (r *upstreamRepository) UpsertUpstreamGroup(ctx context.Context, g *service.UpstreamGroup) error {
	if g.ID > 0 {
		err := r.db.QueryRowContext(ctx,
			`UPDATE upstream_groups
			    SET name = $2, rate_multiplier = $3, peak_rate_multiplier = $4,
			        peak_start = NULLIF($5, ''), peak_end = NULLIF($6, ''),
			        source = $7, note = $8, updated_at = now()
			  WHERE id = $1 AND upstream_id = $9 AND deleted_at IS NULL
			  RETURNING created_at, updated_at`,
			g.ID, g.Name, g.RateMultiplier, g.PeakRateMultiplier, g.PeakStart, g.PeakEnd,
			g.Source, g.Note, g.UpstreamID,
		).Scan(&g.CreatedAt, &g.UpdatedAt)
		if err == sql.ErrNoRows {
			return service.ErrUpstreamGroupNotFound
		}
		if err != nil {
			return fmt.Errorf("update upstream group: %w", err)
		}
		return nil
	}

	var existingID int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM upstream_groups
		  WHERE upstream_id = $1 AND lower(name) = lower($2) AND deleted_at IS NULL`,
		g.UpstreamID, g.Name).Scan(&existingID)
	switch {
	case err == nil:
		g.ID = existingID
		return r.UpsertUpstreamGroup(ctx, g)
	case err != sql.ErrNoRows:
		return fmt.Errorf("lookup upstream group: %w", err)
	}

	if err := r.db.QueryRowContext(ctx,
		`INSERT INTO upstream_groups
		   (upstream_id, name, rate_multiplier, peak_rate_multiplier, peak_start, peak_end, source, note)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, $8)
		 RETURNING id, created_at, updated_at`,
		g.UpstreamID, g.Name, g.RateMultiplier, g.PeakRateMultiplier, g.PeakStart, g.PeakEnd,
		g.Source, g.Note,
	).Scan(&g.ID, &g.CreatedAt, &g.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return service.ErrUpstreamGroupNotFound
		}
		return fmt.Errorf("create upstream group: %w", err)
	}
	return nil
}

// DeleteUpstreamGroup 软删分组；台账的 upstream_group_id 由 FK 保持但语义上不再引用。
func (r *upstreamRepository) DeleteUpstreamGroup(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete upstream group: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`UPDATE upstream_model_prices SET upstream_group_id = NULL, updated_at = now()
		  WHERE upstream_group_id = $1 AND deleted_at IS NULL`, id); err != nil {
		return fmt.Errorf("detach upstream group prices: %w", err)
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE upstream_groups SET deleted_at = now(), updated_at = now()
		  WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("delete upstream group: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrUpstreamGroupNotFound
	}
	return tx.Commit()
}

// ---------- upstream model prices ----------

const upstreamModelPriceColumns = `
	id, upstream_id, upstream_group_id, upstream_model, model_name, platform, billing_mode, currency,
	input_price_per_token, output_price_per_token, cache_read_price_per_token, cache_write_price_per_token,
	per_request_price, rate_multiplier, source, COALESCE(source_url, ''), effective_at, COALESCE(note, ''),
	created_at, updated_at`

func scanUpstreamModelPrice(scanner interface{ Scan(...any) error }) (*service.UpstreamModelPrice, error) {
	p := &service.UpstreamModelPrice{}
	if err := scanner.Scan(
		&p.ID, &p.UpstreamID, &p.UpstreamGroupID, &p.UpstreamModel, &p.ModelName, &p.Platform,
		&p.BillingMode, &p.Currency,
		&p.InputPricePerToken, &p.OutputPricePerToken, &p.CacheReadPricePerToken, &p.CacheWritePricePerToken,
		&p.PerRequestPrice, &p.RateMultiplier, &p.Source, &p.SourceURL, &p.EffectiveAt, &p.Note,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return p, nil
}

// ListUpstreamModelPrices 列出某上游的台账。
func (r *upstreamRepository) ListUpstreamModelPrices(ctx context.Context, upstreamID int64) ([]service.UpstreamModelPrice, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+upstreamModelPriceColumns+`
		   FROM upstream_model_prices
		  WHERE upstream_id = $1 AND deleted_at IS NULL
		  ORDER BY lower(model_name), lower(upstream_model)`, upstreamID)
	if err != nil {
		return nil, fmt.Errorf("list upstream model prices: %w", err)
	}
	defer rows.Close()
	return scanUpstreamModelPriceRows(rows)
}

// ListAllUpstreamModelPrices 列出全部台账（比价用）。
func (r *upstreamRepository) ListAllUpstreamModelPrices(ctx context.Context) ([]service.UpstreamModelPrice, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+upstreamModelPriceColumns+`
		   FROM upstream_model_prices
		  WHERE deleted_at IS NULL
		  ORDER BY lower(model_name), lower(upstream_model)`)
	if err != nil {
		return nil, fmt.Errorf("list all upstream model prices: %w", err)
	}
	defer rows.Close()
	return scanUpstreamModelPriceRows(rows)
}

func scanUpstreamModelPriceRows(rows *sql.Rows) ([]service.UpstreamModelPrice, error) {
	list := make([]service.UpstreamModelPrice, 0, 16)
	for rows.Next() {
		p, err := scanUpstreamModelPrice(rows)
		if err != nil {
			return nil, fmt.Errorf("scan upstream model price: %w", err)
		}
		list = append(list, *p)
	}
	return list, rows.Err()
}

// GetUpstreamModelPrice 按 ID 查询台账条目。
func (r *upstreamRepository) GetUpstreamModelPrice(ctx context.Context, id int64) (*service.UpstreamModelPrice, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+upstreamModelPriceColumns+`
		   FROM upstream_model_prices WHERE id = $1 AND deleted_at IS NULL`, id)
	p, err := scanUpstreamModelPrice(row)
	if err == sql.ErrNoRows {
		return nil, service.ErrUpstreamModelPriceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get upstream model price: %w", err)
	}
	return p, nil
}

// UpsertUpstreamModelPrice 按 ID 或 (upstream_id, group, upstream_model) 幂等写入台账。
func (r *upstreamRepository) UpsertUpstreamModelPrice(ctx context.Context, p *service.UpstreamModelPrice) error {
	if p.ID > 0 {
		err := r.db.QueryRowContext(ctx,
			`UPDATE upstream_model_prices
			    SET upstream_group_id = $2, upstream_model = $3, model_name = $4, platform = $5,
			        billing_mode = $6, currency = $7,
			        input_price_per_token = $8, output_price_per_token = $9,
			        cache_read_price_per_token = $10, cache_write_price_per_token = $11,
			        per_request_price = $12, rate_multiplier = $13, source = $14,
			        source_url = NULLIF($15, ''), effective_at = $16, note = $17, updated_at = now()
			  WHERE id = $1 AND upstream_id = $18 AND deleted_at IS NULL
			  RETURNING created_at, updated_at`,
			p.ID, p.UpstreamGroupID, p.UpstreamModel, p.ModelName, p.Platform,
			p.BillingMode, p.Currency,
			p.InputPricePerToken, p.OutputPricePerToken,
			p.CacheReadPricePerToken, p.CacheWritePricePerToken,
			p.PerRequestPrice, p.RateMultiplier, p.Source,
			p.SourceURL, p.EffectiveAt, p.Note, p.UpstreamID,
		).Scan(&p.CreatedAt, &p.UpdatedAt)
		if err == sql.ErrNoRows {
			return service.ErrUpstreamModelPriceNotFound
		}
		if err != nil {
			return fmt.Errorf("update upstream model price: %w", err)
		}
		return nil
	}

	var existingID int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM upstream_model_prices
		  WHERE upstream_id = $1
		    AND COALESCE(upstream_group_id, 0) = COALESCE($2::bigint, 0)
		    AND upstream_model = $3
		    AND deleted_at IS NULL`,
		p.UpstreamID, p.UpstreamGroupID, p.UpstreamModel).Scan(&existingID)
	switch {
	case err == nil:
		p.ID = existingID
		return r.UpsertUpstreamModelPrice(ctx, p)
	case err != sql.ErrNoRows:
		return fmt.Errorf("lookup upstream model price: %w", err)
	}

	if err := r.db.QueryRowContext(ctx,
		`INSERT INTO upstream_model_prices
		   (upstream_id, upstream_group_id, upstream_model, model_name, platform, billing_mode, currency,
		    input_price_per_token, output_price_per_token, cache_read_price_per_token, cache_write_price_per_token,
		    per_request_price, rate_multiplier, source, source_url, effective_at, note)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NULLIF($15, ''), $16, $17)
		 RETURNING id, created_at, updated_at`,
		p.UpstreamID, p.UpstreamGroupID, p.UpstreamModel, p.ModelName, p.Platform,
		p.BillingMode, p.Currency,
		p.InputPricePerToken, p.OutputPricePerToken, p.CacheReadPricePerToken, p.CacheWritePricePerToken,
		p.PerRequestPrice, p.RateMultiplier, p.Source, p.SourceURL, p.EffectiveAt, p.Note,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return fmt.Errorf("create upstream model price: %w", err)
	}
	return nil
}

// DeleteUpstreamModelPrice 软删台账条目。
func (r *upstreamRepository) DeleteUpstreamModelPrice(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE upstream_model_prices SET deleted_at = now(), updated_at = now()
		  WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("delete upstream model price: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrUpstreamModelPriceNotFound
	}
	return nil
}

// ---------- group upstream links ----------

// groupUpstreamLinkSelect 关联行投影：本站分组、所属上游、上游分组的展示字段一次 JOIN 取全，
// 列表与按 ID 回读共用，避免逐行二次查询。
const groupUpstreamLinkSelect = `
	SELECT l.id, l.group_id, g.name, COALESCE(g.platform, ''), g.rate_multiplier,
	       ug.upstream_id, u.name, ug.id, ug.name, ug.rate_multiplier,
	       COALESCE(l.note, ''), l.created_at, l.updated_at
	  FROM group_upstream_group_links l
	  JOIN groups g ON g.id = l.group_id AND g.deleted_at IS NULL
	  JOIN upstream_groups ug ON ug.id = l.upstream_group_id AND ug.deleted_at IS NULL
	  JOIN upstreams u ON u.id = ug.upstream_id AND u.deleted_at IS NULL`

func scanGroupUpstreamLink(scanner interface{ Scan(...any) error }) (*service.GroupUpstreamLink, error) {
	link := &service.GroupUpstreamLink{}
	if err := scanner.Scan(
		&link.ID, &link.GroupID, &link.GroupName, &link.Platform, &link.GroupRateMultiplier,
		&link.UpstreamID, &link.UpstreamName, &link.UpstreamGroupID, &link.UpstreamGroupName,
		&link.UpstreamRateMultiplier, &link.Note, &link.CreatedAt, &link.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return link, nil
}

// ListGroupUpstreamLinks 列出全部有效供货关联，按本站分组/上游/上游分组名排序。
func (r *upstreamRepository) ListGroupUpstreamLinks(ctx context.Context) ([]service.GroupUpstreamLink, error) {
	rows, err := r.db.QueryContext(ctx, groupUpstreamLinkSelect+`
		 WHERE l.deleted_at IS NULL
		 ORDER BY lower(g.name), lower(u.name), lower(ug.name)`)
	if err != nil {
		return nil, fmt.Errorf("list group upstream links: %w", err)
	}
	defer rows.Close()

	list := make([]service.GroupUpstreamLink, 0, 8)
	for rows.Next() {
		link, scanErr := scanGroupUpstreamLink(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan group upstream link: %w", scanErr)
		}
		list = append(list, *link)
	}
	return list, rows.Err()
}

// updateGroupUpstreamLinkNote 覆盖已有有效关联的 note；无有效行时返回 sql.ErrNoRows 由调用方决定插入。
func updateGroupUpstreamLinkNote(ctx context.Context, tx *sql.Tx, in service.GroupUpstreamLinkInput) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx,
		`UPDATE group_upstream_group_links SET note = NULLIF($3, ''), updated_at = now()
		  WHERE group_id = $1 AND upstream_group_id = $2 AND deleted_at IS NULL
		  RETURNING id`, in.GroupID, in.UpstreamGroupID, in.Note).Scan(&id)
	return id, err
}

// UpsertGroupUpstreamLink 建立或更新供货关联；同一 (本站分组, 上游分组) 只保留一条有效记录。
func (r *upstreamRepository) UpsertGroupUpstreamLink(ctx context.Context, in service.GroupUpstreamLinkInput) (*service.GroupUpstreamLink, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin upsert group upstream link: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 两端都必须存在且未软删；任一端缺失即视为关联目标不存在（同时避免裸 FK 报错泄漏表结构）。
	var exists int64
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM groups WHERE id = $1 AND deleted_at IS NULL`, in.GroupID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrGroupUpstreamLinkNotFound
		}
		return nil, fmt.Errorf("check group for link: %w", err)
	}
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM upstream_groups WHERE id = $1 AND deleted_at IS NULL`, in.UpstreamGroupID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrGroupUpstreamLinkNotFound
		}
		return nil, fmt.Errorf("check upstream group for link: %w", err)
	}

	linkID, err := updateGroupUpstreamLinkNote(ctx, tx, in)
	if err == sql.ErrNoRows {
		// 无有效关联：新建；并发插入撞唯一索引时回滚到保存点后改为更新，保证幂等。
		// 语句失败会让整个事务进入 aborted 状态，因此必须先建保存点才能安全回退。
		if _, err := tx.ExecContext(ctx, `SAVEPOINT group_upstream_link_upsert`); err != nil {
			return nil, fmt.Errorf("savepoint group upstream link: %w", err)
		}
		insertErr := tx.QueryRowContext(ctx,
			`INSERT INTO group_upstream_group_links (group_id, upstream_group_id, note)
			 VALUES ($1, $2, NULLIF($3, ''))
			 RETURNING id`, in.GroupID, in.UpstreamGroupID, in.Note).Scan(&linkID)
		if insertErr != nil {
			if !isUniqueViolation(insertErr) {
				return nil, fmt.Errorf("create group upstream link: %w", insertErr)
			}
			if _, err := tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT group_upstream_link_upsert`); err != nil {
				return nil, fmt.Errorf("rollback group upstream link savepoint: %w", err)
			}
			if linkID, err = updateGroupUpstreamLinkNote(ctx, tx, in); err != nil {
				return nil, fmt.Errorf("update group upstream link after conflict: %w", err)
			}
		}
	} else if err != nil {
		return nil, fmt.Errorf("update group upstream link: %w", err)
	}

	link, err := scanGroupUpstreamLink(tx.QueryRowContext(ctx,
		groupUpstreamLinkSelect+` WHERE l.deleted_at IS NULL AND l.id = $1`, linkID))
	if err != nil {
		return nil, fmt.Errorf("load group upstream link: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit group upstream link: %w", err)
	}
	return link, nil
}

// DeleteGroupUpstreamLink 软删关联；关联已是软删状态时按不存在处理。
func (r *upstreamRepository) DeleteGroupUpstreamLink(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE group_upstream_group_links SET deleted_at = now(), updated_at = now()
		  WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("delete group upstream link: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrGroupUpstreamLinkNotFound
	}
	return nil
}
