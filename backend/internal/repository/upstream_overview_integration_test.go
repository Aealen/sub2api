//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// TestUpstreamOverviewAggregatesIntegration 在真实 Postgres 上验证上游总览的三个聚合查询。
//
// 核心断言：流水必须按「本站分组 × 请求实际使用账号所属的上游」归属。
// 同一个本站分组由两家上游供货时，各家只拿到自己账号产生的流水——
// 既不重复计数（总额不变），也不把一家的流水串到另一家头上。
func TestUpstreamOverviewAggregatesIntegration(t *testing.T) {
	ctx := context.Background()
	repo := &upstreamRepository{db: integrationDB}

	suffix := time.Now().UnixNano()
	var (
		upstreamA, upstreamB      int64
		upstreamGroupA1, groupA2  int64
		upstreamGroupB1           int64
		siteGroup, unlinkedGroup  int64
		accountA, accountB        int64
		userID, apiKeyID          int64
	)

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = integrationDB.ExecContext(cleanupCtx, `DELETE FROM usage_logs WHERE group_id = $1`, siteGroup)
		_, _ = integrationDB.ExecContext(cleanupCtx, `DELETE FROM users WHERE id = $1`, userID)
		_, _ = integrationDB.ExecContext(cleanupCtx,
			`DELETE FROM accounts WHERE id = ANY($1)`, pq.Array([]int64{accountA, accountB}))
		_, _ = integrationDB.ExecContext(cleanupCtx,
			`DELETE FROM groups WHERE id = ANY($1)`, pq.Array([]int64{siteGroup, unlinkedGroup}))
		_, _ = integrationDB.ExecContext(cleanupCtx,
			`DELETE FROM upstreams WHERE id = ANY($1)`, pq.Array([]int64{upstreamA, upstreamB}))
	})

	insertReturningID := func(query string, args ...any) int64 {
		t.Helper()
		var id int64
		require.NoError(t, integrationDB.QueryRowContext(ctx, query, args...).Scan(&id), "insert: %s", query)
		return id
	}

	upstreamA = insertReturningID(`INSERT INTO upstreams (name, status) VALUES ($1, 'active') RETURNING id`,
		fmt.Sprintf("it-uo-a-%d", suffix))
	upstreamB = insertReturningID(`INSERT INTO upstreams (name, status) VALUES ($1, 'active') RETURNING id`,
		fmt.Sprintf("it-uo-b-%d", suffix))

	upstreamGroupA1 = insertReturningID(
		`INSERT INTO upstream_groups (upstream_id, name, rate_multiplier, source) VALUES ($1, $2, $3, 'manual') RETURNING id`,
		upstreamA, "特惠", 0.12)
	groupA2 = insertReturningID(
		`INSERT INTO upstream_groups (upstream_id, name, rate_multiplier, source) VALUES ($1, $2, $3, 'manual') RETURNING id`,
		upstreamA, "标准", 0.30)
	upstreamGroupB1 = insertReturningID(
		`INSERT INTO upstream_groups (upstream_id, name, rate_multiplier, source) VALUES ($1, $2, $3, 'manual') RETURNING id`,
		upstreamB, "经济", 0.20)

	// 一个被两家上游供货的本站分组，一个完全没有关联的本站分组
	siteGroup = insertReturningID(
		`INSERT INTO groups (name, platform, rate_multiplier) VALUES ($1, 'openai', $2) RETURNING id`,
		fmt.Sprintf("it-uo-g1-%d", suffix), 0.16)
	unlinkedGroup = insertReturningID(
		`INSERT INTO groups (name, platform, rate_multiplier) VALUES ($1, 'openai', $2) RETURNING id`,
		fmt.Sprintf("it-uo-g2-%d", suffix), 0.25)

	// 同一本站分组关联两家上游的三个上游分组（含同上游两个分组）
	for _, upstreamGroupID := range []int64{upstreamGroupA1, groupA2, upstreamGroupB1} {
		insertReturningID(
			`INSERT INTO group_upstream_group_links (group_id, upstream_group_id) VALUES ($1, $2) RETURNING id`,
			siteGroup, upstreamGroupID)
	}

	accountA = insertReturningID(
		`INSERT INTO accounts (name, platform, type, status, upstream_id) VALUES ($1, 'openai', 'apikey', 'active', $2) RETURNING id`,
		fmt.Sprintf("it-uo-acc-a-%d", suffix), upstreamA)
	accountB = insertReturningID(
		`INSERT INTO accounts (name, platform, type, status, upstream_id) VALUES ($1, 'openai', 'apikey', 'active', $2) RETURNING id`,
		fmt.Sprintf("it-uo-acc-b-%d", suffix), upstreamB)

	// 两个账号都绑定到同一个本站分组，用于校验「组合下可服务账号数」
	for _, accountID := range []int64{accountA, accountB} {
		_, err := integrationDB.ExecContext(ctx,
			`INSERT INTO account_groups (account_id, group_id, priority) VALUES ($1, $2, 50)`, accountID, siteGroup)
		require.NoError(t, err)
	}

	userID = insertReturningID(
		`INSERT INTO users (email, password_hash) VALUES ($1, 'x') RETURNING id`,
		fmt.Sprintf("it-uo-%d@example.test", suffix))
	apiKeyID = insertReturningID(
		`INSERT INTO api_keys (user_id, key, name) VALUES ($1, $2, $3) RETURNING id`,
		userID, fmt.Sprintf("sk-it-uo-%d", suffix), "it-uo")

	// 流水：A 家账号消费 1 + 2，B 家账号消费 3（同一本站分组）
	insertUsage := func(accountID int64, amount float64) {
		t.Helper()
		_, err := integrationDB.ExecContext(ctx, `
			INSERT INTO usage_logs
				(user_id, api_key_id, account_id, group_id, model,
				 total_cost, actual_cost, account_stats_cost, account_rate_multiplier, created_at)
			VALUES ($1, $2, $3, $4, 'gpt-test', $5, $5, NULL, 1.0, now() - interval '1 day')`,
			userID, apiKeyID, accountID, siteGroup, amount)
		require.NoError(t, err)
	}
	insertUsage(accountA, 1.0)
	insertUsage(accountA, 2.0)
	insertUsage(accountB, 3.0)

	t.Run("流水按账号所属上游归属且不重复计数", func(t *testing.T) {
		rows, err := repo.ListGroupUpstreamFlow(ctx, []int{7})
		require.NoError(t, err)

		revenueByUpstream := map[int64]float64{}
		totalRevenue := 0.0
		for _, row := range rows {
			if row.GroupID != siteGroup || row.WindowDays != 7 {
				continue
			}
			revenueByUpstream[row.UpstreamID] += row.Revenue
			totalRevenue += row.Revenue
			// 成本口径：account_stats_cost 为 NULL 时回退 total_cost，再乘账号倍率 1.0
			require.InDelta(t, row.Revenue, row.Cost, 0.000001, "成本应与收入一致（倍率 1.0）")
		}

		require.Len(t, revenueByUpstream, 2, "同一分组的两家上游都应各占一行")
		require.InDelta(t, 3.0, revenueByUpstream[upstreamA], 0.000001, "A 家应只拿到自己账号的 1+2")
		require.InDelta(t, 3.0, revenueByUpstream[upstreamB], 0.000001, "B 家应只拿到自己账号的 3")
		require.InDelta(t, 6.0, totalRevenue, 0.000001, "总额不应重复放大")
	})

	t.Run("未关联分组只包含没有关联的本站分组", func(t *testing.T) {
		groups, err := repo.ListUnlinkedGroups(ctx)
		require.NoError(t, err)

		found := map[int64]bool{}
		for _, group := range groups {
			found[group.GroupID] = true
		}
		require.True(t, found[unlinkedGroup], "未关联分组应出现在列表里")
		require.False(t, found[siteGroup], "已建立关联的分组不应出现")
	})

	t.Run("上游分组全集带上游名与倍率", func(t *testing.T) {
		options, err := repo.ListUpstreamGroupOptions(ctx)
		require.NoError(t, err)

		byID := map[int64]struct {
			upstreamID   int64
			upstreamName string
			rate         *float64
		}{}
		for _, option := range options {
			byID[option.ID] = struct {
				upstreamID   int64
				upstreamName string
				rate         *float64
			}{option.UpstreamID, option.UpstreamName, option.RateMultiplier}
		}

		entryA1, ok := byID[upstreamGroupA1]
		require.True(t, ok, "上游分组 A1 应出现在全集里")
		require.Equal(t, upstreamA, entryA1.upstreamID)
		require.Contains(t, entryA1.upstreamName, "it-uo-a-")
		require.NotNil(t, entryA1.rate)
		require.InDelta(t, 0.12, *entryA1.rate, 0.000001)

		_, ok = byID[upstreamGroupB1]
		require.True(t, ok, "上游分组 B1 应出现在全集里")
	})

	t.Run("组合下可服务账号数按本站分组与上游归属", func(t *testing.T) {
		counts, err := repo.ListGroupUpstreamBoundAccountCounts(ctx)
		require.NoError(t, err)

		got := map[int64]int{}
		for _, count := range counts {
			if count.GroupID != siteGroup {
				continue
			}
			got[count.UpstreamID] = count.AccountCount
		}

		require.Equal(t, 1, got[upstreamA], "A 家在该分组下绑定 1 个账号")
		require.Equal(t, 1, got[upstreamB], "B 家在该分组下绑定 1 个账号")
	})
}
