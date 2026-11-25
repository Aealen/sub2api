package service

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"
)

// TestDerivedMarginPercent 覆盖推导毛利率的边界：正常值、无售价、进价缺失/为 0、负毛利。
func TestDerivedMarginPercent(t *testing.T) {
	cases := []struct {
		name     string
		sellRate float64
		costRate *float64
		want     *float64
	}{
		{name: "正常值 0.16/0.12", sellRate: 0.16, costRate: ptrFloat(0.12), want: ptrFloat(25)},
		{name: "两位小数舍入 0.3/0.17", sellRate: 0.3, costRate: ptrFloat(0.17), want: ptrFloat(43.33)},
		{name: "零毛利 sellRate=costRate", sellRate: 0.2, costRate: ptrFloat(0.2), want: ptrFloat(0)},
		{name: "sellRate=0", sellRate: 0, costRate: ptrFloat(0.12), want: nil},
		{name: "sellRate 为负", sellRate: -0.16, costRate: ptrFloat(0.12), want: nil},
		{name: "costRate=nil", sellRate: 0.16, costRate: nil, want: nil},
		{name: "costRate=0", sellRate: 0.16, costRate: ptrFloat(0), want: nil},
		{name: "costRate 为负", sellRate: 0.16, costRate: ptrFloat(-0.12), want: nil},
		{name: "负毛利 costRate>sellRate", sellRate: 0.1, costRate: ptrFloat(0.2), want: ptrFloat(-100)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DerivedMarginPercent(tc.sellRate, tc.costRate)
			assertFloatPtr(t, tc.want, got)
		})
	}
}

// TestActualMarginPercent 覆盖实际毛利率的边界：正常值、零收入、负收入、成本高于收入。
func TestActualMarginPercent(t *testing.T) {
	cases := []struct {
		name    string
		revenue float64
		cost    float64
		want    *float64
	}{
		{name: "正常值 1230.5/946.2", revenue: 1230.5, cost: 946.2, want: ptrFloat(23.1)},
		{name: "两位小数舍入 3/1", revenue: 3, cost: 1, want: ptrFloat(66.67)},
		{name: "零成本", revenue: 100, cost: 0, want: ptrFloat(100)},
		{name: "revenue=0", revenue: 0, cost: 10, want: nil},
		{name: "revenue 为负", revenue: -10, cost: 10, want: nil},
		{name: "负毛利 cost>revenue", revenue: 100, cost: 150, want: ptrFloat(-50)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ActualMarginPercent(tc.revenue, tc.cost)
			assertFloatPtr(t, tc.want, got)
		})
	}
}

// TestGetOverviewEmptySlicesAreArrays 空数据集时各切片必须序列化成 []，不能是 null。
func TestGetOverviewEmptySlicesAreArrays(t *testing.T) {
	svc := NewUpstreamService(newFakeUpstreamRepo(), nil)

	result, err := svc.GetOverview(context.Background())
	if err != nil {
		t.Fatalf("GetOverview returned error: %v", err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal overview: %v", err)
	}
	payload := string(raw)
	for _, want := range []string{
		`"windows":[7,30]`,
		`"upstreams":[]`,
		`"upstream_groups":[]`,
		`"links":[]`,
		`"unlinked_groups":[]`,
		`"flow":[]`,
	} {
		if !strings.Contains(payload, want) {
			t.Errorf("payload missing %s, got %s", want, payload)
		}
	}
}

// TestGetOverviewAssemblesDisplayRows 校验毛利率、可服务账号数与预警标记的拼装结果。
func TestGetOverviewAssemblesDisplayRows(t *testing.T) {
	repo := newFakeUpstreamRepo()
	repo.upstreams = []Upstream{{ID: 2, Name: "PPDog", Status: UpstreamStatusActive}}
	repo.accounts = []UpstreamAccountRow{
		{ID: 1, UpstreamID: 2, Status: "active", UpstreamBalance: ptrFloat(57.05)},
		{ID: 2, UpstreamID: 2, Status: "active"},
	}
	repo.alerting = map[int64]bool{2: true}
	repo.links = []GroupUpstreamLink{{
		ID:                     1,
		GroupID:                7,
		GroupName:              "GPT-特惠",
		Platform:               "openai",
		GroupRateMultiplier:    0.16,
		UpstreamID:             2,
		UpstreamName:           "PPDog",
		UpstreamGroupID:        11,
		UpstreamGroupName:      "特惠",
		UpstreamRateMultiplier: ptrFloat(0.12),
	}}
	repo.boundCounts = []GroupUpstreamBoundCount{{GroupID: 7, UpstreamID: 2, AccountCount: 2}}
	repo.groupOptions = []UpstreamGroupOption{{
		ID: 11, UpstreamID: 2, UpstreamName: "PPDog", Name: "特惠", RateMultiplier: ptrFloat(0.12),
	}}
	repo.flow = []GroupUpstreamFlowRow{{
		GroupID: 7, GroupName: "GPT-特惠", UpstreamID: 2, UpstreamName: "PPDog",
		WindowDays: 7, Revenue: 1230.5, Cost: 946.2,
	}}

	result, err := NewUpstreamService(repo, nil).GetOverview(context.Background())
	if err != nil {
		t.Fatalf("GetOverview returned error: %v", err)
	}

	if len(result.Upstreams) != 1 {
		t.Fatalf("expected 1 upstream card, got %d", len(result.Upstreams))
	}
	card := result.Upstreams[0]
	if !card.BalanceAlerting {
		t.Error("expected balance_alerting=true from accounts.extra flag")
	}
	if card.SiteBalance != 57.05 {
		t.Errorf("site_balance = %v, want 57.05", card.SiteBalance)
	}
	if card.GroupNames == nil {
		t.Error("group_names must not be nil")
	}

	if len(result.UpstreamGroups) != 1 {
		t.Fatalf("expected 1 upstream group option, got %d", len(result.UpstreamGroups))
	}
	opt := result.UpstreamGroups[0]
	if opt.ID != 11 || opt.UpstreamID != 2 || opt.UpstreamName != "PPDog" || opt.Name != "特惠" {
		t.Errorf("unexpected upstream group option: %+v", opt)
	}
	if opt.RateMultiplier == nil || math.Abs(*opt.RateMultiplier-0.12) > 1e-9 {
		t.Errorf("rate_multiplier = %v, want 0.12", opt.RateMultiplier)
	}

	if len(result.Links) != 1 {
		t.Fatalf("expected 1 link row, got %d", len(result.Links))
	}
	link := result.Links[0]
	if link.DerivedMarginPercent == nil || math.Abs(*link.DerivedMarginPercent-25) > 1e-9 {
		t.Errorf("derived_margin_pct = %v, want 25", link.DerivedMarginPercent)
	}
	if link.BoundAccountCount != 2 {
		t.Errorf("bound_account_count = %d, want 2", link.BoundAccountCount)
	}

	if len(result.Flow) != 1 {
		t.Fatalf("expected 1 flow row, got %d", len(result.Flow))
	}
	flow := result.Flow[0]
	if flow.Profit != 284.3 {
		t.Errorf("profit = %v, want 284.3", flow.Profit)
	}
	if flow.MarginPct == nil || math.Abs(*flow.MarginPct-23.1) > 1e-9 {
		t.Errorf("margin_pct = %v, want 23.1", flow.MarginPct)
	}
}

func assertFloatPtr(t *testing.T, want, got *float64) {
	t.Helper()
	if want == nil || got == nil {
		if want != nil || got != nil {
			t.Fatalf("got %v, want %v", derefFloat(got), derefFloat(want))
		}
		return
	}
	if math.Abs(*want-*got) > 1e-9 {
		t.Fatalf("got %v, want %v", *got, *want)
	}
}
