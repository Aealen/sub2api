package repository

import (
	"context"
	"fmt"

	"github.com/lib/pq"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 经营概览所需的只读聚合查询。
//
// 与 upstream_repo.go 的既有聚合完全解耦：这里只新增查询，不修改
// ListUpstreamOverview / ListGroupUpstreamLinks 的 SQL 与扫描逻辑。

// ListUpstreamBalanceAlerting 返回当前处于余额预警状态的上游集合。
//
// 不新增阈值、不重算预警：直接复用预警系统已写入 accounts.extra 的
// balance_alert_active 布尔标记（见服务层 upstream_balance_service.go），
// 保证「页面显示预警」与「实际发过预警邮件」是同一个事实。
func (r *upstreamRepository) ListUpstreamBalanceAlerting(ctx context.Context) (map[int64]bool, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT upstream_id
		   FROM accounts
		  WHERE upstream_id IS NOT NULL
		    AND deleted_at IS NULL
		    AND (extra ->> 'balance_alert_active') = 'true'`)
	if err != nil {
		return nil, fmt.Errorf("list upstream balance alerting: %w", err)
	}
	defer rows.Close()

	flags := make(map[int64]bool, 8)
	for rows.Next() {
		var upstreamID int64
		if err := rows.Scan(&upstreamID); err != nil {
			return nil, fmt.Errorf("scan upstream balance alerting: %w", err)
		}
		flags[upstreamID] = true
	}
	return flags, rows.Err()
}

// ListGroupUpstreamBoundAccountCounts 该「本站分组 × 上游」组合下可服务的账号数。
//
// 单条 SQL 聚合（无 N+1）：一个本站分组可能关联同一上游的多个上游分组，
// 因此按 (group_id, upstream_id) 分组并用 COUNT(DISTINCT a.id) 去重，
// 账号数不会随关联条数放大。
//
// 依赖列与索引：group_upstream_group_links(group_id) / (upstream_group_id) 走
// idx_group_upstream_group_links_group / _upstream_group（迁移 254，均带 deleted_at IS NULL）；
// account_groups 按 group_id 走 idx_account_groups_group_id（迁移 001）；
// accounts.upstream_id 走 idx_accounts_upstream_id（迁移 253，WHERE deleted_at IS NULL）。
func (r *upstreamRepository) ListGroupUpstreamBoundAccountCounts(ctx context.Context) ([]service.GroupUpstreamBoundCount, error) {
	query := `
		SELECT l.group_id, ug.upstream_id, COUNT(DISTINCT a.id) AS account_count
		  FROM group_upstream_group_links l
		  JOIN upstream_groups ug ON ug.id = l.upstream_group_id AND ug.deleted_at IS NULL
		  JOIN upstreams u ON u.id = ug.upstream_id AND u.deleted_at IS NULL
		  JOIN groups g ON g.id = l.group_id AND g.deleted_at IS NULL
		  JOIN account_groups ag ON ag.group_id = l.group_id
		  JOIN accounts a ON a.id = ag.account_id
		                 AND a.deleted_at IS NULL
		                 AND a.upstream_id = ug.upstream_id
		 WHERE l.deleted_at IS NULL
		 GROUP BY l.group_id, ug.upstream_id`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list group upstream bound accounts: %w", err)
	}
	defer rows.Close()

	list := make([]service.GroupUpstreamBoundCount, 0, 8)
	for rows.Next() {
		var item service.GroupUpstreamBoundCount
		if err := rows.Scan(&item.GroupID, &item.UpstreamID, &item.AccountCount); err != nil {
			return nil, fmt.Errorf("scan group upstream bound account: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// ListUpstreamGroupOptions 全部有效上游分组（含所属上游名），供前端建立供货关联时选择。
//
// 单条 SQL 查全（无 N+1）：上游与上游分组任一软删都不算有效选项，
// 与 ListGroupUpstreamLinks / ListUnlinkedGroups 的有效性口径保持一致。
// 按上游名、分组名排序，前端下拉可直接渲染。
//
// 依赖列与索引：upstream_groups(upstream_id, lower(name)) 走 idx_upstream_groups_name_active
// （迁移 253，部分索引 WHERE deleted_at IS NULL）；upstreams 按主键 JOIN。
func (r *upstreamRepository) ListUpstreamGroupOptions(ctx context.Context) ([]service.UpstreamGroupOption, error) {
	query := `
		SELECT ug.id, ug.upstream_id, u.name, ug.name, ug.rate_multiplier
		  FROM upstream_groups ug
		  JOIN upstreams u ON u.id = ug.upstream_id AND u.deleted_at IS NULL
		 WHERE ug.deleted_at IS NULL
		 ORDER BY lower(u.name), lower(ug.name), ug.id`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list upstream group options: %w", err)
	}
	defer rows.Close()

	list := make([]service.UpstreamGroupOption, 0, 16)
	for rows.Next() {
		var item service.UpstreamGroupOption
		if err := rows.Scan(&item.ID, &item.UpstreamID, &item.UpstreamName, &item.Name, &item.RateMultiplier); err != nil {
			return nil, fmt.Errorf("scan upstream group option: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// ListUnlinkedGroups 尚未建立任何有效供货关联的本站分组。
//
// 判定口径与 ListGroupUpstreamLinks 一致（关联、上游分组、上游任一软删都不算有效关联），
// 避免出现「links 里看不到、unlinked_groups 里也不提示」的盲区。
//
// 依赖列与索引：group_upstream_group_links(group_id) 走 idx_group_upstream_group_links_group
// （迁移 254，部分索引 WHERE deleted_at IS NULL）；upstream_groups / upstreams 按主键 JOIN。
func (r *upstreamRepository) ListUnlinkedGroups(ctx context.Context) ([]service.UpstreamUnlinkedGroup, error) {
	query := `
		SELECT g.id, g.name, COALESCE(g.platform, ''), g.rate_multiplier
		  FROM groups g
		 WHERE g.deleted_at IS NULL
		   AND NOT EXISTS (
		         SELECT 1
		           FROM group_upstream_group_links l
		           JOIN upstream_groups ug ON ug.id = l.upstream_group_id AND ug.deleted_at IS NULL
		           JOIN upstreams u ON u.id = ug.upstream_id AND u.deleted_at IS NULL
		          WHERE l.group_id = g.id AND l.deleted_at IS NULL
		       )
		 ORDER BY lower(g.name)`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list unlinked groups: %w", err)
	}
	defer rows.Close()

	list := make([]service.UpstreamUnlinkedGroup, 0, 8)
	for rows.Next() {
		var item service.UpstreamUnlinkedGroup
		if err := rows.Scan(&item.GroupID, &item.GroupName, &item.Platform, &item.GroupRateMultiplier); err != nil {
			return nil, fmt.Errorf("scan unlinked group: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// ListGroupUpstreamFlow 分组 × 上游分组的收入与成本聚合（窗口内）。
//
// 归属口径（务必与 usage_logs 的事实来源一致）：
//  1. 分组归属用 usage_logs.group_id：该列是下单时落库的唯一归属，每条流水只有一个
//     分组，天然不会跨组重复计数（不用 account_groups 兜底，因为一个账号可绑多个分组，
//     会造成同一笔消费在多个分组里各算一次）。
//  2. 上游归属用 ul.account_id → accounts.upstream_id：流水实际用了哪个账号，就归它
//     所属的那家上游。同一分组由多家上游供货时，各家只拿到自己账号产生的流水，
//     每条流水恰好归属一家上游，不会在多家上游之间重复计数。
//  3. 只统计 group_id 与 upstream_id 均非空的流水：无分组归属（NULL）的流水无法落到
//     「本站分组 × 上游」行上，直接排除（同时避免扫描 NULL 到 int64 失败）。
//  4. 两个窗口用 CROSS JOIN 展开，窗口内用 created_at >= now() - interval 过滤。
//  5. 不对 groups / accounts / upstreams 追加软删过滤：流水是既成事实，分组或账号
//     事后被软删不应让历史收入凭空消失，否则本报表与账单对不上。
//
// 依赖列与索引：
//   - usage_logs(group_id, created_at)：走 idx_usage_logs_group_created_at_not_null
//     （迁移 062，部分索引 WHERE group_id IS NOT NULL）；
//   - usage_logs(account_id)：走 idx_usage_logs_account_id（迁移 001）连 accounts；
//   - accounts(upstream_id)：走 idx_accounts_upstream_id（迁移 253，部分索引 WHERE deleted_at IS NULL）。
//
// 收入/成本表达式与 usage_log_repo_stats.go 的账号成本口径完全一致，不做改写。
func (r *upstreamRepository) ListGroupUpstreamFlow(ctx context.Context, windows []int) ([]service.GroupUpstreamFlowRow, error) {
	query := `
		WITH windows(days) AS (
		    SELECT unnest($1::int[]) AS days
		)
		SELECT ul.group_id, g.name, a.upstream_id, u.name, w.days,
		       COALESCE(SUM(ul.actual_cost), 0) AS revenue,
		       COALESCE(SUM(COALESCE(ul.account_stats_cost, ul.total_cost)
		                    * COALESCE(ul.account_rate_multiplier, 1)), 0) AS cost
		  FROM windows w
		  CROSS JOIN usage_logs ul
		  JOIN accounts a ON a.id = ul.account_id
		  JOIN groups g ON g.id = ul.group_id
		  JOIN upstreams u ON u.id = a.upstream_id
		 WHERE ul.created_at >= now() - make_interval(days => w.days)
		   AND ul.group_id IS NOT NULL
		   AND a.upstream_id IS NOT NULL
		 GROUP BY ul.group_id, g.name, a.upstream_id, u.name, w.days
		 ORDER BY lower(g.name), a.upstream_id, w.days`
	rows, err := r.db.QueryContext(ctx, query, pq.Array(intSliceToInt64(windows)))
	if err != nil {
		return nil, fmt.Errorf("list group upstream flow: %w", err)
	}
	defer rows.Close()

	list := make([]service.GroupUpstreamFlowRow, 0, 16)
	for rows.Next() {
		var item service.GroupUpstreamFlowRow
		if err := rows.Scan(
			&item.GroupID, &item.GroupName, &item.UpstreamID, &item.UpstreamName,
			&item.WindowDays, &item.Revenue, &item.Cost,
		); err != nil {
			return nil, fmt.Errorf("scan group upstream flow: %w", err)
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// intSliceToInt64 把窗口天数转成 pq.Array 直接支持的 []int64。
func intSliceToInt64(in []int) []int64 {
	out := make([]int64, len(in))
	for i, v := range in {
		out[i] = int64(v)
	}
	return out
}
