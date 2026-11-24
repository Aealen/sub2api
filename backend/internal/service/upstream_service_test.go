package service

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"
)

// ---------- fake repository ----------

type fakeUpstreamRepo struct {
	upstreams []Upstream
	accounts  []UpstreamAccountRow
	groups    []UpstreamGroup
	prices    []UpstreamModelPrice
	nextID    int64
}

func newFakeUpstreamRepo() *fakeUpstreamRepo {
	return &fakeUpstreamRepo{nextID: 1}
}

func (r *fakeUpstreamRepo) id() int64 {
	id := r.nextID
	r.nextID++
	return id
}

func (r *fakeUpstreamRepo) CreateUpstream(_ context.Context, u *Upstream) error {
	u.ID = r.id()
	u.CreatedAt = time.Now()
	u.UpdatedAt = u.CreatedAt
	r.upstreams = append(r.upstreams, *u)
	return nil
}

func (r *fakeUpstreamRepo) GetUpstreamByID(_ context.Context, id int64) (*Upstream, error) {
	for i := range r.upstreams {
		if r.upstreams[i].ID == id {
			u := r.upstreams[i]
			return &u, nil
		}
	}
	return nil, ErrUpstreamNotFound
}

func (r *fakeUpstreamRepo) GetUpstreamByName(_ context.Context, name string) (*Upstream, error) {
	for i := range r.upstreams {
		if strings.EqualFold(r.upstreams[i].Name, name) {
			u := r.upstreams[i]
			return &u, nil
		}
	}
	return nil, ErrUpstreamNotFound
}

func (r *fakeUpstreamRepo) GetUpstreamByBaseURL(_ context.Context, baseURL string) (*Upstream, error) {
	for i := range r.upstreams {
		if r.upstreams[i].BaseURL == baseURL {
			u := r.upstreams[i]
			return &u, nil
		}
	}
	return nil, ErrUpstreamNotFound
}

func (r *fakeUpstreamRepo) UpdateUpstream(_ context.Context, u *Upstream) error {
	for i := range r.upstreams {
		if r.upstreams[i].ID == u.ID {
			r.upstreams[i] = *u
			return nil
		}
	}
	return ErrUpstreamNotFound
}

func (r *fakeUpstreamRepo) DeleteUpstream(_ context.Context, id int64) error {
	for i := range r.upstreams {
		if r.upstreams[i].ID == id {
			r.upstreams = append(r.upstreams[:i], r.upstreams[i+1:]...)
			return nil
		}
	}
	return ErrUpstreamNotFound
}

func (r *fakeUpstreamRepo) ListUpstreams(_ context.Context) ([]Upstream, error) {
	return append([]Upstream(nil), r.upstreams...), nil
}

func (r *fakeUpstreamRepo) ListUpstreamOverview(_ context.Context) ([]UpstreamOverview, error) {
	out := make([]UpstreamOverview, 0, len(r.upstreams))
	for _, u := range r.upstreams {
		row := UpstreamOverview{Upstream: u}
		for _, a := range r.accounts {
			if a.UpstreamID != u.ID {
				continue
			}
			row.AccountTotal++
			if a.Status == "active" {
				row.AccountActive++
			}
			// 与生产聚合一致：整站余额取第一个非空值（同站各 Key 余额相同）
			if row.SiteBalance == 0 && a.UpstreamBalance != nil {
				row.SiteBalance = *a.UpstreamBalance
			}
		}
		for _, p := range r.prices {
			if p.UpstreamID == u.ID {
				row.ModelPriceCount++
			}
		}
		out = append(out, row)
	}
	return out, nil
}

func (r *fakeUpstreamRepo) ListUnassignedAccountHosts(_ context.Context) ([]UnassignedAccountHost, error) {
	return nil, nil
}

func (r *fakeUpstreamRepo) AssignAccountsToUpstream(_ context.Context, accountIDs []int64, upstreamID int64) error {
	for i := range r.accounts {
		for _, id := range accountIDs {
			if r.accounts[i].ID == id {
				r.accounts[i].UpstreamID = upstreamID
			}
		}
	}
	return nil
}

func (r *fakeUpstreamRepo) ListUpstreamAccounts(_ context.Context, upstreamID int64) ([]UpstreamAccountRow, error) {
	out := make([]UpstreamAccountRow, 0, len(r.accounts))
	for _, a := range r.accounts {
		if a.UpstreamID == upstreamID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *fakeUpstreamRepo) ListAllAssignedAccounts(_ context.Context) ([]UpstreamAccountRow, error) {
	out := make([]UpstreamAccountRow, 0, len(r.accounts))
	for _, a := range r.accounts {
		if a.UpstreamID != 0 {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *fakeUpstreamRepo) ListUpstreamGroups(_ context.Context, upstreamID int64) ([]UpstreamGroup, error) {
	out := make([]UpstreamGroup, 0, len(r.groups))
	for _, g := range r.groups {
		if g.UpstreamID == upstreamID {
			out = append(out, g)
		}
	}
	return out, nil
}

func (r *fakeUpstreamRepo) GetUpstreamGroup(_ context.Context, id int64) (*UpstreamGroup, error) {
	for i := range r.groups {
		if r.groups[i].ID == id {
			g := r.groups[i]
			return &g, nil
		}
	}
	return nil, ErrUpstreamGroupNotFound
}

func (r *fakeUpstreamRepo) UpsertUpstreamGroup(_ context.Context, g *UpstreamGroup) error {
	for i := range r.groups {
		if g.ID > 0 && r.groups[i].ID == g.ID {
			r.groups[i] = *g
			return nil
		}
		if g.ID == 0 && r.groups[i].UpstreamID == g.UpstreamID &&
			strings.EqualFold(r.groups[i].Name, g.Name) {
			g.ID = r.groups[i].ID
			r.groups[i] = *g
			return nil
		}
	}
	g.ID = r.id()
	r.groups = append(r.groups, *g)
	return nil
}

func (r *fakeUpstreamRepo) DeleteUpstreamGroup(_ context.Context, id int64) error {
	for i := range r.groups {
		if r.groups[i].ID == id {
			r.groups = append(r.groups[:i], r.groups[i+1:]...)
			return nil
		}
	}
	return ErrUpstreamGroupNotFound
}

func (r *fakeUpstreamRepo) ListUpstreamModelPrices(_ context.Context, upstreamID int64) ([]UpstreamModelPrice, error) {
	out := make([]UpstreamModelPrice, 0, len(r.prices))
	for _, p := range r.prices {
		if p.UpstreamID == upstreamID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakeUpstreamRepo) ListAllUpstreamModelPrices(_ context.Context) ([]UpstreamModelPrice, error) {
	return append([]UpstreamModelPrice(nil), r.prices...), nil
}

func (r *fakeUpstreamRepo) GetUpstreamModelPrice(_ context.Context, id int64) (*UpstreamModelPrice, error) {
	for i := range r.prices {
		if r.prices[i].ID == id {
			p := r.prices[i]
			return &p, nil
		}
	}
	return nil, ErrUpstreamModelPriceNotFound
}

func (r *fakeUpstreamRepo) UpsertUpstreamModelPrice(_ context.Context, p *UpstreamModelPrice) error {
	for i := range r.prices {
		if r.prices[i].UpstreamID == p.UpstreamID &&
			equalInt64Ptr(r.prices[i].UpstreamGroupID, p.UpstreamGroupID) &&
			r.prices[i].UpstreamModel == p.UpstreamModel {
			p.ID = r.prices[i].ID
			r.prices[i] = *p
			return nil
		}
	}
	p.ID = r.id()
	r.prices = append(r.prices, *p)
	return nil
}

func (r *fakeUpstreamRepo) DeleteUpstreamModelPrice(_ context.Context, id int64) error {
	for i := range r.prices {
		if r.prices[i].ID == id {
			r.prices = append(r.prices[:i], r.prices[i+1:]...)
			return nil
		}
	}
	return ErrUpstreamModelPriceNotFound
}

func equalInt64Ptr(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// ---------- helpers ----------

func testBillingService(t *testing.T) *BillingService {
	t.Helper()
	stub := newStubPricingServiceFromJSON(t, `{
		"deepseek-v4-pro": {"litellm_provider": "deepseek", "mode": "chat",
			"input_cost_per_token": 4e-07, "output_cost_per_token": 1.6e-06},
		"claude-sonnet-4-5": {"litellm_provider": "anthropic", "mode": "chat",
			"input_cost_per_token": 3e-06, "output_cost_per_token": 1.5e-05}
	}`)
	return &BillingService{pricingService: stub}
}

func ptrFloat(v float64) *float64 { return &v }
func ptrInt64(v int64) *int64     { return &v }

func derefFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

// ---------- tests ----------

func TestNormalizeUpstreamBaseURL(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"https://api.hualong.ai", "https://api.hualong.ai"},
		{"https://api.hualong.ai/", "https://api.hualong.ai"},
		{"https://api.hualong.ai/v1", "https://api.hualong.ai"},
		{"http://API.HuaLong.ai:8080/v1/", "http://api.hualong.ai:8080"},
		{"api.hualong.ai", "https://api.hualong.ai"},
		{"  https://ppdog.example.com/v1  ", "https://ppdog.example.com"},
		{"", ""},
		{"ftp://weird.example.com", "https://weird.example.com"},
	}
	for _, tc := range cases {
		if got := NormalizeUpstreamBaseURL(tc.raw); got != tc.want {
			t.Fatalf("NormalizeUpstreamBaseURL(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestCompareModelsLedgerOverridesDerived(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUpstreamRepo()
	repo.upstreams = []Upstream{
		{ID: 1, Name: "华龙", Status: UpstreamStatusActive},
		{ID: 2, Name: "勿忘", Status: UpstreamStatusActive},
	}
	repo.accounts = []UpstreamAccountRow{
		{ID: 10, UpstreamID: 1, Name: "华龙-C", Platform: PlatformAnthropic, Status: "active",
			RateMultiplier: 0.48, ModelMapping: map[string]string{"claude-sonnet-4-5": "claude-sonnet-4-5-hl"}},
		{ID: 20, UpstreamID: 2, Name: "勿忘-C", Platform: PlatformAnthropic, Status: "active",
			RateMultiplier: 0.08, ModelMapping: map[string]string{"claude-sonnet-4-5": "claude-sonnet-4-5-ww"}},
	}
	// 华龙台账报价：高于勿忘的推导价，用于验证「最低价」标记落在推导侧
	repo.prices = []UpstreamModelPrice{
		{ID: 1, UpstreamID: 1, UpstreamModel: "claude-sonnet-4-5-hl", ModelName: "claude-sonnet-4-5",
			InputPricePerToken: ptrFloat(3e-07), OutputPricePerToken: ptrFloat(1.2e-06),
			Source: UpstreamPriceSourceManual},
	}

	svc := NewUpstreamService(repo, testBillingService(t))
	result, err := svc.CompareModels(ctx, "", "")
	if err != nil {
		t.Fatalf("CompareModels error: %v", err)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result.Rows))
	}
	row := result.Rows[0]
	if row.ModelName != "claude-sonnet-4-5" {
		t.Fatalf("unexpected row model: %s", row.ModelName)
	}
	if len(row.Cells) != 2 {
		t.Fatalf("expected 2 cells, got %d", len(row.Cells))
	}

	byUpstream := map[int64]UpstreamCompareCell{}
	for _, cell := range row.Cells {
		byUpstream[cell.UpstreamID] = cell
	}
	ledger := byUpstream[1]
	if ledger.Source != upstreamCompareSourceLedger {
		t.Fatalf("华龙 cell source = %s, want ledger", ledger.Source)
	}
	if ledger.InputPricePerToken == nil || *ledger.InputPricePerToken != 3e-07 {
		t.Fatalf("华龙 input price = %v, want 3e-07", derefFloat(ledger.InputPricePerToken))
	}
	derived := byUpstream[2]
	if derived.Source != upstreamCompareSourceDerived {
		t.Fatalf("勿忘 cell source = %s, want derived", derived.Source)
	}
	// 目录价 3e-06 × 0.08
	if derived.InputPricePerToken == nil || math.Abs(*derived.InputPricePerToken-3e-06*0.08) > 1e-18 {
		t.Fatalf("勿忘 input price = %v, want %v", derefFloat(derived.InputPricePerToken), 3e-06*0.08)
	}
	if row.CheapestUpstreamID == nil || *row.CheapestUpstreamID != 2 {
		t.Fatalf("cheapest upstream = %v, want 2", row.CheapestUpstreamID)
	}
}

func TestCompareModelsDerivedUsesLowestAccountMultiplier(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUpstreamRepo()
	repo.upstreams = []Upstream{{ID: 1, Name: "PPDog", Status: UpstreamStatusActive}}
	repo.accounts = []UpstreamAccountRow{
		{ID: 11, UpstreamID: 1, Name: "PPDog-A", Platform: PlatformAnthropic, Status: "active",
			RateMultiplier: 0.25, ModelMapping: map[string]string{"claude-sonnet-4-5": "claude-sonnet-4-5-a"}},
		{ID: 12, UpstreamID: 1, Name: "PPDog-B", Platform: PlatformAnthropic, Status: "active",
			RateMultiplier: 0.12, ModelMapping: map[string]string{"claude-sonnet-4-5": "claude-sonnet-4-5-b"}},
	}
	svc := NewUpstreamService(repo, testBillingService(t))
	result, err := svc.CompareModels(ctx, "", "")
	if err != nil {
		t.Fatalf("CompareModels error: %v", err)
	}
	row := result.Rows[0]
	if len(row.Cells) != 1 {
		t.Fatalf("expected 1 cell, got %d", len(row.Cells))
	}
	cell := row.Cells[0]
	if cell.AccountID == nil || *cell.AccountID != 12 {
		t.Fatalf("expected lowest-multiplier account 12, got %v", cell.AccountID)
	}
	// 目录价 3e-06 × 0.12
	if cell.InputPricePerToken == nil || math.Abs(*cell.InputPricePerToken-3e-06*0.12) > 1e-18 {
		t.Fatalf("input price = %v, want %v", derefFloat(cell.InputPricePerToken), 3e-06*0.12)
	}
}

func TestCompareModelsSkipsUpstreamWithoutMatchingModel(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUpstreamRepo()
	repo.upstreams = []Upstream{
		{ID: 1, Name: "华龙", Status: UpstreamStatusActive},
		{ID: 2, Name: "勿忘", Status: UpstreamStatusActive},
	}
	repo.accounts = []UpstreamAccountRow{
		{ID: 10, UpstreamID: 1, Name: "华龙-DS", Platform: PlatformOpenAI, Status: "active",
			RateMultiplier: 0.48, ModelMapping: map[string]string{"deepseek-v4-pro": "hl-ds"}},
	}
	svc := NewUpstreamService(repo, testBillingService(t))
	result, err := svc.CompareModels(ctx, "", "")
	if err != nil {
		t.Fatalf("CompareModels error: %v", err)
	}
	if len(result.Rows[0].Cells) != 1 {
		t.Fatalf("expected only 1 comparable cell, got %d", len(result.Rows[0].Cells))
	}
	if result.Rows[0].CheapestUpstreamID != nil {
		t.Fatal("single-cell row must not mark a cheapest upstream")
	}
}

func TestImportModelPricesResolvesAliasAndMarksUnmatched(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUpstreamRepo()
	repo.upstreams = []Upstream{{ID: 1, Name: "华龙", Status: UpstreamStatusActive}}
	repo.accounts = []UpstreamAccountRow{
		{ID: 10, UpstreamID: 1, Name: "华龙-DS", Platform: PlatformOpenAI, Status: "active",
			RateMultiplier: 0.48, ModelMapping: map[string]string{"deepseek-v4-pro": "hl-deepseek-v4-pro"}},
	}
	svc := NewUpstreamService(repo, nil)

	items := []UpstreamModelPriceInput{
		{UpstreamModel: "hl-deepseek-v4-pro", InputPricePerToken: ptrFloat(4e-07), UpstreamGroupName: "默认"},
		{UpstreamModel: "unknown-model", InputPricePerToken: ptrFloat(1e-06)},
	}
	result, err := svc.ImportModelPrices(ctx, 1, items, UpstreamPriceSourceCrawler)
	if err != nil {
		t.Fatalf("ImportModelPrices error: %v", err)
	}
	if result.Upserted != 2 {
		t.Fatalf("upserted = %d, want 2", result.Upserted)
	}
	if len(result.Unmatched) != 1 || result.Unmatched[0] != "unknown-model" {
		t.Fatalf("unmatched = %v, want [unknown-model]", result.Unmatched)
	}

	prices, _ := repo.ListUpstreamModelPrices(ctx, 1)
	var resolved, unmatched *UpstreamModelPrice
	for i := range prices {
		switch prices[i].UpstreamModel {
		case "hl-deepseek-v4-pro":
			resolved = &prices[i]
		case "unknown-model":
			unmatched = &prices[i]
		}
	}
	if resolved == nil || resolved.ModelName != "deepseek-v4-pro" {
		t.Fatalf("alias resolution failed: %+v", resolved)
	}
	if resolved.Source != UpstreamPriceSourceCrawler {
		t.Fatalf("source = %s, want crawler", resolved.Source)
	}
	if resolved.UpstreamGroupID == nil {
		t.Fatal("group name should be auto-created and linked")
	}
	if unmatched == nil || unmatched.ModelName != "unknown-model" {
		t.Fatalf("unmatched model should fall back to upstream name: %+v", unmatched)
	}

	// 幂等：同样输入再导入一次不应新增行
	before := len(repo.prices)
	if _, err := svc.ImportModelPrices(ctx, 1, items, UpstreamPriceSourceCrawler); err != nil {
		t.Fatalf("second import error: %v", err)
	}
	if len(repo.prices) != before {
		t.Fatalf("second import created duplicates: before=%d after=%d", before, len(repo.prices))
	}
}

func TestCreateUpstreamRejectsDuplicateNameAndBaseURL(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUpstreamRepo()
	repo.upstreams = []Upstream{{ID: 1, Name: "华龙", BaseURL: "https://api.hualong.ai", Status: UpstreamStatusActive}}
	svc := NewUpstreamService(repo, nil)

	if _, err := svc.CreateUpstream(ctx, UpstreamInput{Name: "华龙"}); err != ErrUpstreamExists {
		t.Fatalf("duplicate name err = %v, want ErrUpstreamExists", err)
	}
	if _, err := svc.CreateUpstream(ctx, UpstreamInput{Name: "另一家", BaseURL: "https://api.hualong.ai/v1"}); err != ErrUpstreamBaseURLExists {
		t.Fatalf("duplicate base_url err = %v, want ErrUpstreamBaseURLExists", err)
	}
	created, err := svc.CreateUpstream(ctx, UpstreamInput{Name: "勿忘", BaseURL: "https://api.wuwang.ai/v1/"})
	if err != nil {
		t.Fatalf("create upstream error: %v", err)
	}
	if created.BaseURL != "https://api.wuwang.ai" {
		t.Fatalf("base_url not normalized: %q", created.BaseURL)
	}
	if created.Status != UpstreamStatusActive {
		t.Fatalf("default status = %q, want active", created.Status)
	}
}

func TestCompareModelsGroupColumns(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUpstreamRepo()
	repo.upstreams = []Upstream{{ID: 1, Name: "华龙", Status: UpstreamStatusActive}}
	repo.groups = []UpstreamGroup{{ID: 100, UpstreamID: 1, Name: "default", RateMultiplier: ptrFloat(0.48)}, {ID: 101, UpstreamID: 1, Name: "vip", RateMultiplier: ptrFloat(0.25)}}
	repo.accounts = []UpstreamAccountRow{
		{ID: 10, UpstreamID: 1, Name: "华龙-DS", Platform: PlatformAnthropic, Status: "active",
			RateMultiplier: 0.48, ModelMapping: map[string]string{"claude-sonnet-4-5": "hl-cs"}},
	}
	// 分组台账：vip 组挂一条「价格即最终价」的台账
	repo.prices = []UpstreamModelPrice{
		{ID: 1, UpstreamID: 1, UpstreamGroupID: ptrInt64(101), UpstreamModel: "hl-cs-vip", ModelName: "claude-sonnet-4-5",
			InputPricePerToken: ptrFloat(1e-6), Source: UpstreamPriceSourceManual},
	}

	svc := NewUpstreamService(repo, testBillingService(t))
	result, err := svc.CompareModels(ctx, "", "")
	if err != nil {
		t.Fatalf("CompareModels error: %v", err)
	}
	// 列：default(分组) + vip(分组) + 综合
	if len(result.Columns) != 3 {
		t.Fatalf("columns = %d, want 3", len(result.Columns))
	}
	row := result.Rows[0]
	if len(row.Cells) != 3 {
		t.Fatalf("cells = %d, want 3", len(row.Cells))
	}
	byKey := map[string]UpstreamCompareCell{}
	for _, cell := range row.Cells {
		key := ledgerGroupKey(cell.GroupID)
		byKey[key] = cell
	}
	// vip 台账价
	vip := byKey["g:101"]
	if vip.Source != upstreamCompareSourceLedger || vip.InputPricePerToken == nil || *vip.InputPricePerToken != 1e-6 {
		t.Fatalf("vip cell wrong: %+v", vip)
	}
	// default 无台账 → 目录价 × 分组倍率
	def := byKey["g:100"]
	if def.Source != upstreamCompareSourceDerived || def.InputPricePerToken == nil || math.Abs(*def.InputPricePerToken-3e-06*0.48) > 1e-18 {
		t.Fatalf("default cell wrong: %+v", def)
	}
	// 综合列：账号最低倍率
	derived := byKey["-"]
	if derived.Source != upstreamCompareSourceDerived || derived.AccountID == nil || *derived.AccountID != 10 {
		t.Fatalf("derived cell wrong: %+v", derived)
	}
	if row.CheapestUpstreamID == nil || *row.CheapestUpstreamID != 1 || row.CheapestGroupID == nil || *row.CheapestGroupID != 101 {
		t.Fatalf("cheapest = %v/%v, want upstream 1 group 101(vip 台账列)", row.CheapestUpstreamID, row.CheapestGroupID)
	}
}

func TestCompareModelsGroupRateDerivationOnly(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUpstreamRepo()
	repo.upstreams = []Upstream{{ID: 1, Name: "PPDog", Status: UpstreamStatusActive}}
	repo.groups = []UpstreamGroup{{ID: 200, UpstreamID: 1, Name: "特惠", RateMultiplier: ptrFloat(0.05)}}
	repo.accounts = []UpstreamAccountRow{
		{ID: 30, UpstreamID: 1, Name: "PPDog-A", Platform: PlatformAnthropic, Status: "active",
			RateMultiplier: 0.12, ModelMapping: map[string]string{"claude-sonnet-4-5": "pd-cs"}},
	}
	// 无台账：分组列纯按分组倍率推导，综合列按账号倍率推导
	svc := NewUpstreamService(repo, testBillingService(t))
	result, err := svc.CompareModels(ctx, "", "claude-sonnet-4-5")
	if err != nil {
		t.Fatalf("CompareModels error: %v", err)
	}
	if len(result.Columns) != 2 || result.Columns[0].Kind != UpstreamCompareColumnGroup {
		t.Fatalf("columns = %+v, want group + derived columns", result.Columns)
	}
	row := result.Rows[0]
	if len(row.Cells) != 2 {
		t.Fatalf("cells = %d, want 2", len(row.Cells))
	}
	var cell UpstreamCompareCell
	for _, c := range row.Cells {
		if c.GroupID != nil && *c.GroupID == 200 {
			cell = c
		}
	}
	if cell.GroupID == nil {
		t.Fatal("group cell missing")
	}
	if cell.InputPricePerToken == nil || math.Abs(*cell.InputPricePerToken-3e-06*0.05) > 1e-18 {
		t.Fatalf("cell price = %v, want %v", derefFloat(cell.InputPricePerToken), 3e-06*0.05)
	}
}
