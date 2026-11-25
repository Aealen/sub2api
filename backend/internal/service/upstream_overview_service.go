package service

import (
	"context"
	"math"
	"time"
)

// UpstreamOverviewWindows 经营概览固定输出的观察窗口（天）。
//
// 前端按 7 天看近期走势、30 天看月度盘子，窗口集合不做参数化，
// 保证同一份响应里的窗口口径稳定可比。
var UpstreamOverviewWindows = []int{7, 30}

// UpstreamOverviewCard 上游卡片：在 UpstreamOverview 聚合之上补充余额预警标记。
//
// 只做展示投影，不引入阈值字段；balance_alerting 直接读预警系统已写入的
// accounts.extra.balance_alert_active，避免出现第二套预警判断。
type UpstreamOverviewCard struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	Status            string     `json:"status"`
	SiteBalance       float64    `json:"site_balance"`
	BalanceUpdatedAt  *time.Time `json:"balance_updated_at"`
	AccountTotal      int        `json:"account_total"`
	AccountActive     int        `json:"account_active"`
	GroupCount        int        `json:"group_count"`
	ModelPriceCount   int        `json:"model_price_count"`
	RateMultiplierMin *float64   `json:"rate_multiplier_min"`
	RateMultiplierMax *float64   `json:"rate_multiplier_max"`
	GroupNames        []string   `json:"group_names"`
	BalanceAlerting   bool       `json:"balance_alerting"`
}

// UpstreamOverviewLink 本站分组 × 上游分组的供货关联展示行。
type UpstreamOverviewLink struct {
	ID                     int64    `json:"id"`
	GroupID                int64    `json:"group_id"`
	GroupName              string   `json:"group_name"`
	Platform               string   `json:"platform"`
	GroupRateMultiplier    float64  `json:"group_rate_multiplier"`
	UpstreamID             int64    `json:"upstream_id"`
	UpstreamName           string   `json:"upstream_name"`
	UpstreamGroupID        int64    `json:"upstream_group_id"`
	UpstreamGroupName      string   `json:"upstream_group_name"`
	UpstreamRateMultiplier *float64 `json:"upstream_rate_multiplier"`
	// DerivedMarginPercent 由「本站售价倍率 - 上游进价倍率」推导的毛利率（%）。
	DerivedMarginPercent *float64 `json:"derived_margin_pct"`
	// BoundAccountCount 该组合下可服务的账号数：上游归属该组合、且绑定到本站分组的未删账号。
	BoundAccountCount int `json:"bound_account_count"`
}

// UpstreamGroupOption 可供关联的上游分组选项（上游侧全量有效分组）。
//
// 前端建立「本站分组 × 上游分组」关联时需要先选一家上游、再选它的套餐分组；
// 这里一次返回全部有效上游分组并带上游名，避免前端为每个上游各查一次（N+1）。
type UpstreamGroupOption struct {
	ID             int64    `json:"id"`
	UpstreamID     int64    `json:"upstream_id"`
	UpstreamName   string   `json:"upstream_name"`
	Name           string   `json:"name"`
	RateMultiplier *float64 `json:"rate_multiplier"`
}

// UpstreamUnlinkedGroup 尚未建立任何有效供货关联的本站分组。
type UpstreamUnlinkedGroup struct {
	GroupID             int64   `json:"group_id"`
	GroupName           string  `json:"group_name"`
	Platform            string  `json:"platform"`
	GroupRateMultiplier float64 `json:"group_rate_multiplier"`
}

// UpstreamFlowRow 分组 × 上游在某个观察窗口内的收入、成本与毛利。
type UpstreamFlowRow struct {
	GroupID      int64    `json:"group_id"`
	GroupName    string   `json:"group_name"`
	UpstreamID   int64    `json:"upstream_id"`
	UpstreamName string   `json:"upstream_name"`
	WindowDays   int      `json:"window_days"`
	Revenue      float64  `json:"revenue"`
	Cost         float64  `json:"cost"`
	Profit       float64  `json:"profit"`
	MarginPct    *float64 `json:"margin_pct"`
}

// UpstreamOverviewResult 上游经营概览聚合结果。
type UpstreamOverviewResult struct {
	Windows   []int                  `json:"windows"`
	Upstreams []UpstreamOverviewCard `json:"upstreams"`
	// UpstreamGroups 全部有效上游分组，供前端建立供货关联时选择。
	UpstreamGroups []UpstreamGroupOption   `json:"upstream_groups"`
	Links          []UpstreamOverviewLink  `json:"links"`
	UnlinkedGroups []UpstreamUnlinkedGroup `json:"unlinked_groups"`
	Flow           []UpstreamFlowRow       `json:"flow"`
}

// GroupUpstreamBoundCount 「本站分组 × 上游」组合下可服务的账号数（仓储聚合行）。
type GroupUpstreamBoundCount struct {
	GroupID      int64
	UpstreamID   int64
	AccountCount int
}

// GroupUpstreamFlowRow 分组流水窗口聚合原始行（毛利在服务层计算）。
type GroupUpstreamFlowRow struct {
	GroupID      int64
	GroupName    string
	UpstreamID   int64
	UpstreamName string
	WindowDays   int
	Revenue      float64
	Cost         float64
}

// DerivedMarginPercent 由售价倍率与进价倍率推导毛利率（百分比，保留两位小数）。
//
// 语义：sellRate <= 0（无有效售价）、costRate 为 nil（上游未维护倍率）、
// costRate <= 0（进价倍率缺失/非法）时返回 nil，表示「不可比较」而不是「零毛利」，
// 避免把数据缺口渲染成 0% 误导判断。
func DerivedMarginPercent(sellRate float64, costRate *float64) *float64 {
	if sellRate <= 0 || costRate == nil || *costRate <= 0 {
		return nil
	}
	margin := roundTwo((1 - *costRate/sellRate) * 100)
	return &margin
}

// ActualMarginPercent 由窗口内实际收入与成本计算毛利率（百分比，保留两位小数）。
//
// revenue <= 0 时无意义（无消费或有退款冲抵），返回 nil。
func ActualMarginPercent(revenue, cost float64) *float64 {
	if revenue <= 0 {
		return nil
	}
	margin := roundTwo((revenue - cost) / revenue * 100)
	return &margin
}

// roundTwo 四舍五入到两位小数。
func roundTwo(v float64) float64 {
	return math.Round(v*100) / 100
}

// GetOverview 上游经营概览：上游卡片、供货关联、未关联分组与分组流水。
//
// 纯读聚合，不触碰任何计费/扣费/预警写入路径；nil 切片统一补空数组，
// 避免 JSON 输出 null 导致前端渲染崩溃。
func (s *UpstreamService) GetOverview(ctx context.Context) (*UpstreamOverviewResult, error) {
	overviews, err := s.repo.ListUpstreamOverview(ctx)
	if err != nil {
		return nil, err
	}
	alerting, err := s.repo.ListUpstreamBalanceAlerting(ctx)
	if err != nil {
		return nil, err
	}

	cards := make([]UpstreamOverviewCard, 0, len(overviews))
	for _, o := range overviews {
		groupNames := o.GroupNames
		if groupNames == nil {
			groupNames = []string{}
		}
		cards = append(cards, UpstreamOverviewCard{
			ID:                o.ID,
			Name:              o.Name,
			Status:            o.Status,
			SiteBalance:       o.SiteBalance,
			BalanceUpdatedAt:  o.BalanceUpdatedAt,
			AccountTotal:      o.AccountTotal,
			AccountActive:     o.AccountActive,
			GroupCount:        o.GroupCount,
			ModelPriceCount:   o.ModelPriceCount,
			RateMultiplierMin: o.RateMultiplierMin,
			RateMultiplierMax: o.RateMultiplierMax,
			GroupNames:        groupNames,
			BalanceAlerting:   alerting[o.ID],
		})
	}

	groupOptions, err := s.repo.ListUpstreamGroupOptions(ctx)
	if err != nil {
		return nil, err
	}
	if groupOptions == nil {
		groupOptions = []UpstreamGroupOption{}
	}

	links, err := s.repo.ListGroupUpstreamLinks(ctx)
	if err != nil {
		return nil, err
	}
	boundCounts, err := s.repo.ListGroupUpstreamBoundAccountCounts(ctx)
	if err != nil {
		return nil, err
	}
	boundIndex := make(map[groupUpstreamKey]int, len(boundCounts))
	for _, bc := range boundCounts {
		boundIndex[groupUpstreamKey{groupID: bc.GroupID, upstreamID: bc.UpstreamID}] = bc.AccountCount
	}
	linkRows := make([]UpstreamOverviewLink, 0, len(links))
	for _, l := range links {
		linkRows = append(linkRows, UpstreamOverviewLink{
			ID:                     l.ID,
			GroupID:                l.GroupID,
			GroupName:              l.GroupName,
			Platform:               l.Platform,
			GroupRateMultiplier:    l.GroupRateMultiplier,
			UpstreamID:             l.UpstreamID,
			UpstreamName:           l.UpstreamName,
			UpstreamGroupID:        l.UpstreamGroupID,
			UpstreamGroupName:      l.UpstreamGroupName,
			UpstreamRateMultiplier: l.UpstreamRateMultiplier,
			DerivedMarginPercent:   DerivedMarginPercent(l.GroupRateMultiplier, l.UpstreamRateMultiplier),
			BoundAccountCount:      boundIndex[groupUpstreamKey{groupID: l.GroupID, upstreamID: l.UpstreamID}],
		})
	}

	unlinked, err := s.repo.ListUnlinkedGroups(ctx)
	if err != nil {
		return nil, err
	}
	if unlinked == nil {
		unlinked = []UpstreamUnlinkedGroup{}
	}

	flowRaw, err := s.repo.ListGroupUpstreamFlow(ctx, UpstreamOverviewWindows)
	if err != nil {
		return nil, err
	}
	flow := make([]UpstreamFlowRow, 0, len(flowRaw))
	for _, f := range flowRaw {
		flow = append(flow, UpstreamFlowRow{
			GroupID:      f.GroupID,
			GroupName:    f.GroupName,
			UpstreamID:   f.UpstreamID,
			UpstreamName: f.UpstreamName,
			WindowDays:   f.WindowDays,
			Revenue:      roundTwo(f.Revenue),
			Cost:         roundTwo(f.Cost),
			Profit:       roundTwo(f.Revenue - f.Cost),
			MarginPct:    ActualMarginPercent(f.Revenue, f.Cost),
		})
	}

	return &UpstreamOverviewResult{
		Windows:        append([]int(nil), UpstreamOverviewWindows...),
		Upstreams:      cards,
		UpstreamGroups: groupOptions,
		Links:          linkRows,
		UnlinkedGroups: unlinked,
		Flow:           flow,
	}, nil
}

// groupUpstreamKey 「本站分组 × 上游」组合的索引键。
type groupUpstreamKey struct {
	groupID    int64
	upstreamID int64
}
