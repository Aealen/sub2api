package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	// ErrUpstreamNotFound 上游不存在。
	ErrUpstreamNotFound = infraerrors.NotFound("UPSTREAM_NOT_FOUND", "upstream not found")
	// ErrUpstreamExists 上游名称重复（不区分大小写）。
	ErrUpstreamExists = infraerrors.Conflict("UPSTREAM_EXISTS", "upstream name already exists")
	// ErrUpstreamBaseURLExists 同一 base_url 已归属其他上游。
	ErrUpstreamBaseURLExists = infraerrors.Conflict("UPSTREAM_BASE_URL_EXISTS", "upstream base_url already belongs to another upstream")
	// ErrUpstreamGroupNotFound 上游分组不存在。
	ErrUpstreamGroupNotFound = infraerrors.NotFound("UPSTREAM_GROUP_NOT_FOUND", "upstream group not found")
	// ErrUpstreamModelPriceNotFound 上游模型价格台账条目不存在。
	ErrUpstreamModelPriceNotFound = infraerrors.NotFound("UPSTREAM_MODEL_PRICE_NOT_FOUND", "upstream model price not found")
)

// 上游状态与价格来源常量。
const (
	UpstreamStatusActive   = "active"
	UpstreamStatusInactive = "inactive"

	UpstreamPriceSourceManual  = "manual"
	UpstreamPriceSourceImport  = "import"
	UpstreamPriceSourceCrawler = "crawler"
	UpstreamGroupSourceProbe   = "probe"

	// upstreamCompareSourceLedger 台账价优先于自动推导价。
	upstreamCompareSourceLedger = "ledger"
	// upstreamCompareSourceDerived 由「目录价 × 账号倍率」推导。
	upstreamCompareSourceDerived = "derived"
)

// Upstream 上游实体（转售站）。
//
// 账号通过 accounts.upstream_id 归属上游；余额、进价倍率、模型映射仍以账号为事实来源，
// 上游只提供聚合视图与商务元信息，避免出现第二份事实。
type Upstream struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	BaseURL   string     `json:"base_url"`
	Website   string     `json:"website"`
	Contact   string     `json:"contact"`
	Note      string     `json:"note"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-"`
}

// UpstreamInput 上游新建/更新入参。
type UpstreamInput struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Website string `json:"website"`
	Contact string `json:"contact"`
	Note    string `json:"note"`
	Status  string `json:"status"`
}

// UpstreamOverview 上游列表聚合行：余额、账号数、倍率区间与台账规模。
type UpstreamOverview struct {
	Upstream
	AccountTotal      int        `json:"account_total"`
	AccountActive     int        `json:"account_active"`
	BalanceTotal      float64    `json:"balance_total"`
	BalanceUpdatedAt  *time.Time `json:"balance_updated_at"`
	RateMultiplierMin *float64   `json:"rate_multiplier_min"`
	RateMultiplierMax *float64   `json:"rate_multiplier_max"`
	ModelPriceCount   int        `json:"model_price_count"`
	GroupCount        int        `json:"group_count"`
	GroupNames        []string   `json:"group_names"`
}

// UpstreamAccountRow 上游详情中的账号行（轻量投影，不加载完整账号凭证）。
type UpstreamAccountRow struct {
	ID                       int64             `json:"id"`
	UpstreamID               int64             `json:"upstream_id"`
	Name                     string            `json:"name"`
	Platform                 string            `json:"platform"`
	Type                     string            `json:"type"`
	Status                   string            `json:"status"`
	BaseURL                  string            `json:"base_url"`
	RateMultiplier           float64           `json:"rate_multiplier"`
	UpstreamBalance          *float64          `json:"upstream_balance"`
	UpstreamBalanceUpdatedAt *time.Time        `json:"upstream_balance_updated_at"`
	ModelMapping             map[string]string `json:"model_mapping"`
	MappedModelCount         int               `json:"mapped_model_count"`
	GroupIDs                 []int64           `json:"group_ids"`
	GroupNames               []string          `json:"group_names"`
	// ProbedGroupRateMultiplier 上游 /v1/sub2api/billing 探测到的分组倍率（若有）。
	ProbedGroupRateMultiplier *float64 `json:"probed_group_rate_multiplier"`
	// ProbedEffectiveRateMultiplier 探测到的最终生效倍率（含高峰，若有）。
	ProbedEffectiveRateMultiplier *float64 `json:"probed_effective_rate_multiplier"`
	ProbedPeakMultiplier          *float64 `json:"probed_peak_multiplier"`
	ProbedPeakStart               string   `json:"probed_peak_start"`
	ProbedPeakEnd                 string   `json:"probed_peak_end"`
	ProbedObservedAt              string   `json:"probed_observed_at"`
}

// UnassignedAccountHost 尚未归属上游的账号按 base_url host 聚合，供管理端引导归并。
type UnassignedAccountHost struct {
	BaseURL      string   `json:"base_url"`
	Host         string   `json:"host"`
	AccountCount int      `json:"account_count"`
	SampleNames  []string `json:"sample_names"`
	AccountIDs   []int64  `json:"account_ids"`
}

// UpstreamGroup 上游侧分组（转售站内部的分组/套餐及倍率）。
type UpstreamGroup struct {
	ID                 int64     `json:"id"`
	UpstreamID         int64     `json:"upstream_id"`
	Name               string    `json:"name"`
	RateMultiplier     *float64  `json:"rate_multiplier"`
	PeakRateMultiplier *float64  `json:"peak_rate_multiplier"`
	PeakStart          string    `json:"peak_start"`
	PeakEnd            string    `json:"peak_end"`
	Source             string    `json:"source"`
	Note               string    `json:"note"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// UpstreamGroupInput 上游分组新建/更新入参。
type UpstreamGroupInput struct {
	ID                 int64    `json:"id"`
	Name               string   `json:"name"`
	RateMultiplier     *float64 `json:"rate_multiplier"`
	PeakRateMultiplier *float64 `json:"peak_rate_multiplier"`
	PeakStart          string   `json:"peak_start"`
	PeakEnd            string   `json:"peak_end"`
	Source             string   `json:"source"`
	Note               string   `json:"note"`
}

// UpstreamModelPrice 上游模型价格台账条目。
//
// 价格单位统一为 USD per-token（与 billing 目录价、channel_model_pricing 一致）；
// rate_multiplier 非空时有效价 = price × rate_multiplier，为空表示 price 已是最终价。
type UpstreamModelPrice struct {
	ID                      int64      `json:"id"`
	UpstreamID              int64      `json:"upstream_id"`
	UpstreamGroupID         *int64     `json:"upstream_group_id"`
	UpstreamModel           string     `json:"upstream_model"`
	ModelName               string     `json:"model_name"`
	Platform                string     `json:"platform"`
	BillingMode             string     `json:"billing_mode"`
	Currency                string     `json:"currency"`
	InputPricePerToken      *float64   `json:"input_price_per_token"`
	OutputPricePerToken     *float64   `json:"output_price_per_token"`
	CacheReadPricePerToken  *float64   `json:"cache_read_price_per_token"`
	CacheWritePricePerToken *float64   `json:"cache_write_price_per_token"`
	PerRequestPrice         *float64   `json:"per_request_price"`
	RateMultiplier          *float64   `json:"rate_multiplier"`
	Source                  string     `json:"source"`
	SourceURL               string     `json:"source_url"`
	EffectiveAt             *time.Time `json:"effective_at"`
	Note                    string     `json:"note"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// UpstreamModelPriceInput 台账写入入参（上游/分组由路由或分组 id 决定）。
type UpstreamModelPriceInput struct {
	ID              int64  `json:"id"`
	UpstreamGroupID *int64 `json:"upstream_group_id"`
	// UpstreamGroupName 按分组名关联（导入/爬虫推送常用）；找不到时自动创建该分组。
	UpstreamGroupName       string     `json:"upstream_group_name"`
	UpstreamModel           string     `json:"upstream_model"`
	ModelName               string     `json:"model_name"`
	Platform                string     `json:"platform"`
	BillingMode             string     `json:"billing_mode"`
	Currency                string     `json:"currency"`
	InputPricePerToken      *float64   `json:"input_price_per_token"`
	OutputPricePerToken     *float64   `json:"output_price_per_token"`
	CacheReadPricePerToken  *float64   `json:"cache_read_price_per_token"`
	CacheWritePricePerToken *float64   `json:"cache_write_price_per_token"`
	PerRequestPrice         *float64   `json:"per_request_price"`
	RateMultiplier          *float64   `json:"rate_multiplier"`
	Source                  string     `json:"source"`
	SourceURL               string     `json:"source_url"`
	EffectiveAt             *time.Time `json:"effective_at"`
	Note                    string     `json:"note"`
}

// UpstreamPriceImportResult 批量导入结果。
type UpstreamPriceImportResult struct {
	Upserted  int      `json:"upserted"`
	Skipped   int      `json:"skipped"`
	Unmatched []string `json:"unmatched"`
}

// UpstreamBundleImport 上游整包导入（供外部爬虫/脚本一次推送一家上游的完整配置）。
type UpstreamBundleImport struct {
	Upstream    UpstreamInput             `json:"upstream"`
	Groups      []UpstreamGroupInput      `json:"groups"`
	ModelPrices []UpstreamModelPriceInput `json:"model_prices"`
}

// UpstreamCompareColumnKind 比价列类型。
const (
	// UpstreamCompareColumnGroup 上游侧分组列（倍率来自 upstream_groups）。
	UpstreamCompareColumnGroup = "group"
	// UpstreamCompareColumnDerived 综合列：无分组信息时按账号最低倍率推导（含无分组台账）。
	UpstreamCompareColumnDerived = "derived"
)

// UpstreamCompareColumn 比价矩阵列：上游 × 上游侧分组。
//
// 一个上游常有多个分组、多个倍率，只有一列会抹平价差；
// GroupID 为 nil 表示该上游的「综合」列（无分组台账 + 账号最低倍率推导）。
type UpstreamCompareColumn struct {
	UpstreamID     int64    `json:"upstream_id"`
	UpstreamName   string   `json:"upstream_name"`
	GroupID        *int64   `json:"group_id"`
	GroupName      string   `json:"group_name"`
	RateMultiplier *float64 `json:"rate_multiplier"`
	Kind           string   `json:"kind"` // group | derived
}

// UpstreamCompareCell 比价矩阵单元格：某上游某分组对某模型的进价。
type UpstreamCompareCell struct {
	UpstreamID          int64    `json:"upstream_id"`
	GroupID             *int64   `json:"group_id"`
	UpstreamName        string   `json:"upstream_name"`
	InputPricePerToken  *float64 `json:"input_price_per_token"`
	OutputPricePerToken *float64 `json:"output_price_per_token"`
	RateMultiplier      *float64 `json:"rate_multiplier"`
	Source              string   `json:"source"` // ledger | derived
	AccountID           *int64   `json:"account_id,omitempty"`
	AccountName         string   `json:"account_name,omitempty"`
}

// UpstreamCompareRow 比价矩阵行：一个平台标准模型在各上游/分组的进价。
type UpstreamCompareRow struct {
	ModelName                   string                `json:"model_name"`
	Platform                    string                `json:"platform"`
	OfficialInputPricePerToken  *float64              `json:"official_input_price_per_token"`
	OfficialOutputPricePerToken *float64              `json:"official_output_price_per_token"`
	Cells                       []UpstreamCompareCell `json:"cells"`
	// CheapestUpstreamID / CheapestGroupID 进价最低的单元格（仅在有 >=2 个可比单元格时填充）。
	CheapestUpstreamID *int64 `json:"cheapest_upstream_id"`
	CheapestGroupID    *int64 `json:"cheapest_group_id"`
}

// UpstreamCompareResult 跨上游比价结果。
type UpstreamCompareResult struct {
	Upstreams []Upstream              `json:"upstreams"`
	Columns   []UpstreamCompareColumn `json:"columns"`
	Rows      []UpstreamCompareRow    `json:"rows"`
}

// UpstreamDetail 上游详情聚合。
type UpstreamDetail struct {
	Upstream
	Accounts    []UpstreamAccountRow `json:"accounts"`
	Groups      []UpstreamGroup      `json:"groups"`
	ModelPrices []UpstreamModelPrice `json:"model_prices"`
}

// UpstreamRepository 上游管理的数据访问接口（消费者定义）。
type UpstreamRepository interface {
	CreateUpstream(ctx context.Context, u *Upstream) error
	GetUpstreamByID(ctx context.Context, id int64) (*Upstream, error)
	GetUpstreamByName(ctx context.Context, name string) (*Upstream, error)
	GetUpstreamByBaseURL(ctx context.Context, baseURL string) (*Upstream, error)
	UpdateUpstream(ctx context.Context, u *Upstream) error
	DeleteUpstream(ctx context.Context, id int64) error
	ListUpstreams(ctx context.Context) ([]Upstream, error)
	ListUpstreamOverview(ctx context.Context) ([]UpstreamOverview, error)

	ListUnassignedAccountHosts(ctx context.Context) ([]UnassignedAccountHost, error)
	AssignAccountsToUpstream(ctx context.Context, accountIDs []int64, upstreamID int64) error
	ListUpstreamAccounts(ctx context.Context, upstreamID int64) ([]UpstreamAccountRow, error)
	ListAllAssignedAccounts(ctx context.Context) ([]UpstreamAccountRow, error)

	ListUpstreamGroups(ctx context.Context, upstreamID int64) ([]UpstreamGroup, error)
	GetUpstreamGroup(ctx context.Context, id int64) (*UpstreamGroup, error)
	UpsertUpstreamGroup(ctx context.Context, g *UpstreamGroup) error
	DeleteUpstreamGroup(ctx context.Context, id int64) error

	ListUpstreamModelPrices(ctx context.Context, upstreamID int64) ([]UpstreamModelPrice, error)
	ListAllUpstreamModelPrices(ctx context.Context) ([]UpstreamModelPrice, error)
	GetUpstreamModelPrice(ctx context.Context, id int64) (*UpstreamModelPrice, error)
	UpsertUpstreamModelPrice(ctx context.Context, p *UpstreamModelPrice) error
	DeleteUpstreamModelPrice(ctx context.Context, id int64) error
}

// UpstreamService 上游管理与跨上游比价。
//
// 价格事实来源优先级：
//  1. 台账价（upstream_model_prices，人工录入/导入/爬虫推送）—— 代表上游实际报价
//  2. 推导价（目录价 × 该上游支持该模型账号的最低倍率）—— 代表平台当前实际进货成本
//
// 两者并存而非互相覆盖：台账回答「供应商报价多少」，推导回答「我们现在付多少」。
type UpstreamService struct {
	repo           UpstreamRepository
	billingService *BillingService
}

// NewUpstreamService 创建上游管理服务；billingService 可为 nil（测试场景，比价将缺少目录价）。
func NewUpstreamService(repo UpstreamRepository, billingService *BillingService) *UpstreamService {
	return &UpstreamService{repo: repo, billingService: billingService}
}

// NormalizeUpstreamBaseURL 归一化上游地址用于账号归并：只保留 scheme://host。
//
// 不同账号常常写成 `https://api.foo.com`、`https://api.foo.com/`、`https://api.foo.com/v1`
// 三种形态，归并建议必须把它们视为同一上游，否则会提示重复创建。
func NormalizeUpstreamBaseURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	parsed, err := url.Parse(s)
	if err != nil || parsed.Host == "" {
		if err != nil {
			return strings.TrimRight(strings.TrimSpace(raw), "/")
		}
		return ""
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		scheme = "https"
	}
	return scheme + "://" + strings.ToLower(parsed.Host)
}

// ListOverview 上游列表（含余额、账号数、倍率区间与台账规模）。
func (s *UpstreamService) ListOverview(ctx context.Context) ([]UpstreamOverview, error) {
	return s.repo.ListUpstreamOverview(ctx)
}

// GetDetail 上游详情：基本信息 + 账号 + 上游分组 + 台账。
func (s *UpstreamService) GetDetail(ctx context.Context, id int64) (*UpstreamDetail, error) {
	u, err := s.repo.GetUpstreamByID(ctx, id)
	if err != nil {
		return nil, err
	}
	accounts, err := s.repo.ListUpstreamAccounts(ctx, id)
	if err != nil {
		return nil, err
	}
	groups, err := s.repo.ListUpstreamGroups(ctx, id)
	if err != nil {
		return nil, err
	}
	prices, err := s.repo.ListUpstreamModelPrices(ctx, id)
	if err != nil {
		return nil, err
	}
	if accounts == nil {
		accounts = []UpstreamAccountRow{}
	}
	if groups == nil {
		groups = []UpstreamGroup{}
	}
	if prices == nil {
		prices = []UpstreamModelPrice{}
	}
	return &UpstreamDetail{Upstream: *u, Accounts: accounts, Groups: groups, ModelPrices: prices}, nil
}

// CreateUpstream 新建上游；名称与 base_url 都要求唯一（软删后可复用）。
func (s *UpstreamService) CreateUpstream(ctx context.Context, in UpstreamInput) (*Upstream, error) {
	normalized, err := s.validateUpstreamInput(ctx, in, 0)
	if err != nil {
		return nil, err
	}
	u := &Upstream{
		Name:    strings.TrimSpace(in.Name),
		BaseURL: normalized,
		Website: strings.TrimSpace(in.Website),
		Contact: strings.TrimSpace(in.Contact),
		Note:    strings.TrimSpace(in.Note),
		Status:  normalizeUpstreamStatus(in.Status),
	}
	if err := s.repo.CreateUpstream(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

// UpdateUpstream 更新上游元信息。
func (s *UpstreamService) UpdateUpstream(ctx context.Context, id int64, in UpstreamInput) (*Upstream, error) {
	current, err := s.repo.GetUpstreamByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != "" {
		current.Name = strings.TrimSpace(in.Name)
	}
	normalized, err := s.validateUpstreamInput(ctx, UpstreamInput{
		Name:    current.Name,
		BaseURL: in.BaseURL,
	}, id)
	if err != nil {
		return nil, err
	}
	current.BaseURL = normalized
	current.Website = strings.TrimSpace(in.Website)
	current.Contact = strings.TrimSpace(in.Contact)
	current.Note = strings.TrimSpace(in.Note)
	if in.Status != "" {
		current.Status = normalizeUpstreamStatus(in.Status)
	}
	if err := s.repo.UpdateUpstream(ctx, current); err != nil {
		return nil, err
	}
	return current, nil
}

// validateUpstreamInput 校验名称/状态并归一化 base_url，返回归一化后的地址。
func (s *UpstreamService) validateUpstreamInput(ctx context.Context, in UpstreamInput, excludeID int64) (string, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return "", infraerrors.BadRequest("UPSTREAM_NAME_REQUIRED", "upstream name is required")
	}
	if len([]rune(name)) > 100 {
		return "", infraerrors.BadRequest("UPSTREAM_NAME_TOO_LONG", "upstream name must be at most 100 characters")
	}
	if existing, err := s.repo.GetUpstreamByName(ctx, name); err == nil && existing != nil && existing.ID != excludeID {
		return "", ErrUpstreamExists
	} else if err != nil && !errors.Is(err, ErrUpstreamNotFound) {
		return "", err
	}

	normalized := NormalizeUpstreamBaseURL(in.BaseURL)
	if normalized != "" {
		if existing, err := s.repo.GetUpstreamByBaseURL(ctx, normalized); err == nil && existing != nil && existing.ID != excludeID {
			return "", ErrUpstreamBaseURLExists
		} else if err != nil && !errors.Is(err, ErrUpstreamNotFound) {
			return "", err
		}
	}
	return normalized, nil
}

// DeleteUpstream 软删上游；账号的 upstream_id 由 DB 置空，账号本身保留。
func (s *UpstreamService) DeleteUpstream(ctx context.Context, id int64) error {
	return s.repo.DeleteUpstream(ctx, id)
}

// ListUnassignedAccountHosts 未归属账号按 host 聚合（管理端归并提示）。
func (s *UpstreamService) ListUnassignedAccountHosts(ctx context.Context) ([]UnassignedAccountHost, error) {
	hosts, err := s.repo.ListUnassignedAccountHosts(ctx)
	if err != nil {
		return nil, err
	}
	if hosts == nil {
		hosts = []UnassignedAccountHost{}
	}
	return hosts, nil
}

// AssignAccounts 批量把账号挂到上游。
func (s *UpstreamService) AssignAccounts(ctx context.Context, upstreamID int64, accountIDs []int64) error {
	if len(accountIDs) == 0 {
		return infraerrors.BadRequest("UPSTREAM_ACCOUNTS_REQUIRED", "account_ids is required")
	}
	if _, err := s.repo.GetUpstreamByID(ctx, upstreamID); err != nil {
		return err
	}
	return s.repo.AssignAccountsToUpstream(ctx, accountIDs, upstreamID)
}

// UpsertGroup 新建或更新上游侧分组。
func (s *UpstreamService) UpsertGroup(ctx context.Context, upstreamID int64, in UpstreamGroupInput) (*UpstreamGroup, error) {
	if _, err := s.repo.GetUpstreamByID(ctx, upstreamID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, infraerrors.BadRequest("UPSTREAM_GROUP_NAME_REQUIRED", "upstream group name is required")
	}
	source := strings.TrimSpace(in.Source)
	switch source {
	case "":
		source = UpstreamPriceSourceManual
	case UpstreamGroupSourceProbe, UpstreamPriceSourceImport, UpstreamPriceSourceCrawler, UpstreamPriceSourceManual:
	default:
		return nil, infraerrors.BadRequest("UPSTREAM_GROUP_SOURCE_INVALID", "unsupported upstream group source")
	}
	g := &UpstreamGroup{
		ID:                 in.ID,
		UpstreamID:         upstreamID,
		Name:               name,
		RateMultiplier:     in.RateMultiplier,
		PeakRateMultiplier: in.PeakRateMultiplier,
		PeakStart:          strings.TrimSpace(in.PeakStart),
		PeakEnd:            strings.TrimSpace(in.PeakEnd),
		Source:             source,
		Note:               strings.TrimSpace(in.Note),
	}
	if err := s.repo.UpsertUpstreamGroup(ctx, g); err != nil {
		return nil, err
	}
	return g, nil
}

// DeleteGroup 删除上游分组；台账条目的 upstream_group_id 由 DB 置空。
func (s *UpstreamService) DeleteGroup(ctx context.Context, upstreamID, groupID int64) error {
	g, err := s.repo.GetUpstreamGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if g.UpstreamID != upstreamID {
		return ErrUpstreamGroupNotFound
	}
	return s.repo.DeleteUpstreamGroup(ctx, groupID)
}

// UpsertModelPrice 单条台账写入。
func (s *UpstreamService) UpsertModelPrice(ctx context.Context, upstreamID int64, in UpstreamModelPriceInput) (*UpstreamModelPrice, error) {
	if _, err := s.repo.GetUpstreamByID(ctx, upstreamID); err != nil {
		return nil, err
	}
	price, err := s.buildModelPrice(ctx, upstreamID, in, UpstreamPriceSourceManual)
	if err != nil {
		return nil, err
	}
	if _, err := s.applyModelNameFallback(ctx, upstreamID, price); err != nil {
		return nil, err
	}
	groupID, err := s.resolveGroupID(ctx, upstreamID, in.UpstreamGroupID, in.UpstreamGroupName, price.Source)
	if err != nil {
		return nil, err
	}
	price.UpstreamGroupID = groupID
	if err := s.repo.UpsertUpstreamModelPrice(ctx, price); err != nil {
		return nil, err
	}
	return price, nil
}

// DeleteModelPrice 删除台账条目。
func (s *UpstreamService) DeleteModelPrice(ctx context.Context, upstreamID, priceID int64) error {
	price, err := s.repo.GetUpstreamModelPrice(ctx, priceID)
	if err != nil {
		return err
	}
	if price.UpstreamID != upstreamID {
		return ErrUpstreamModelPriceNotFound
	}
	return s.repo.DeleteUpstreamModelPrice(ctx, priceID)
}

// ImportModelPrices 批量导入台账（幂等 upsert）。
//
// model_name 缺省时按该上游账号的 model_mapping 反查（{标准名: 上游名}），
// 反查不到则记入 unmatched 由管理端人工确认，绝不静默丢数据。
func (s *UpstreamService) ImportModelPrices(ctx context.Context, upstreamID int64, in []UpstreamModelPriceInput, source string) (*UpstreamPriceImportResult, error) {
	if _, err := s.repo.GetUpstreamByID(ctx, upstreamID); err != nil {
		return nil, err
	}
	if source != UpstreamPriceSourceCrawler && source != UpstreamPriceSourceImport {
		source = UpstreamPriceSourceImport
	}
	result := &UpstreamPriceImportResult{Unmatched: []string{}}
	for _, item := range in {
		price, buildErr := s.buildModelPrice(ctx, upstreamID, item, source)
		if buildErr != nil {
			result.Skipped++
			continue
		}
		matched, aliasErr := s.applyModelNameFallback(ctx, upstreamID, price)
		if aliasErr != nil {
			return nil, aliasErr
		}
		if !matched {
			result.Unmatched = append(result.Unmatched, price.UpstreamModel)
		}
		groupID, groupErr := s.resolveGroupID(ctx, upstreamID, item.UpstreamGroupID, item.UpstreamGroupName, price.Source)
		if groupErr != nil {
			return nil, groupErr
		}
		price.UpstreamGroupID = groupID
		if err := s.repo.UpsertUpstreamModelPrice(ctx, price); err != nil {
			return nil, err
		}
		result.Upserted++
	}
	return result, nil
}

// ImportBundle 上游整包导入：不存在则创建上游，存在则更新元信息，再导入分组与台账。
func (s *UpstreamService) ImportBundle(ctx context.Context, bundle UpstreamBundleImport, source string) (*Upstream, *UpstreamPriceImportResult, error) {
	existing, err := s.repo.GetUpstreamByName(ctx, strings.TrimSpace(bundle.Upstream.Name))
	if err != nil && !errors.Is(err, ErrUpstreamNotFound) {
		return nil, nil, err
	}
	var (
		u         *Upstream
		createErr error
	)
	if existing != nil {
		u, createErr = s.UpdateUpstream(ctx, existing.ID, bundle.Upstream)
	} else {
		u, createErr = s.CreateUpstream(ctx, bundle.Upstream)
	}
	if createErr != nil {
		return nil, nil, createErr
	}

	// 先落分组（携带倍率等元信息），台账的分组名由 ImportModelPrices 自动关联
	for _, g := range bundle.Groups {
		if _, gErr := s.UpsertGroup(ctx, u.ID, g); gErr != nil {
			return nil, nil, gErr
		}
	}
	result, err := s.ImportModelPrices(ctx, u.ID, bundle.ModelPrices, source)
	if err != nil {
		return nil, nil, err
	}
	return u, result, nil
}

// buildModelPrice 校验并规范化一条台账入参。
func (s *UpstreamService) buildModelPrice(ctx context.Context, upstreamID int64, in UpstreamModelPriceInput, defaultSource string) (*UpstreamModelPrice, error) {
	upstreamModel := strings.TrimSpace(in.UpstreamModel)
	modelName := strings.TrimSpace(in.ModelName)
	if upstreamModel == "" && modelName == "" {
		return nil, infraerrors.BadRequest("UPSTREAM_MODEL_REQUIRED", "upstream_model or model_name is required")
	}
	if upstreamModel == "" {
		upstreamModel = modelName
	}
	billingMode := strings.TrimSpace(in.BillingMode)
	switch billingMode {
	case "":
		billingMode = "token"
	case "token", "per_request", "image", "video":
	default:
		return nil, infraerrors.BadRequest("UPSTREAM_BILLING_MODE_INVALID", "unsupported billing_mode")
	}
	currency := strings.TrimSpace(in.Currency)
	if currency == "" {
		currency = "USD"
	}
	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = defaultSource
	}
	switch source {
	case UpstreamPriceSourceManual, UpstreamPriceSourceImport, UpstreamPriceSourceCrawler:
	default:
		source = defaultSource
	}
	// 单价必须为非负；未填视为不提供该维度价格
	if err := validateNonNegative(in.InputPricePerToken, "input_price_per_token"); err != nil {
		return nil, err
	}
	if err := validateNonNegative(in.OutputPricePerToken, "output_price_per_token"); err != nil {
		return nil, err
	}
	if err := validateNonNegative(in.CacheReadPricePerToken, "cache_read_price_per_token"); err != nil {
		return nil, err
	}
	if err := validateNonNegative(in.CacheWritePricePerToken, "cache_write_price_per_token"); err != nil {
		return nil, err
	}
	if err := validateNonNegative(in.PerRequestPrice, "per_request_price"); err != nil {
		return nil, err
	}
	if err := validateNonNegative(in.RateMultiplier, "rate_multiplier"); err != nil {
		return nil, err
	}
	return &UpstreamModelPrice{
		ID:                      in.ID,
		UpstreamID:              upstreamID,
		UpstreamGroupID:         in.UpstreamGroupID,
		UpstreamModel:           upstreamModel,
		ModelName:               modelName,
		Platform:                strings.TrimSpace(in.Platform),
		BillingMode:             billingMode,
		Currency:                currency,
		InputPricePerToken:      in.InputPricePerToken,
		OutputPricePerToken:     in.OutputPricePerToken,
		CacheReadPricePerToken:  in.CacheReadPricePerToken,
		CacheWritePricePerToken: in.CacheWritePricePerToken,
		PerRequestPrice:         in.PerRequestPrice,
		RateMultiplier:          in.RateMultiplier,
		Source:                  source,
		SourceURL:               strings.TrimSpace(in.SourceURL),
		EffectiveAt:             in.EffectiveAt,
		Note:                    strings.TrimSpace(in.Note),
	}, nil
}

// upstreamModelAlias 构建「上游模型名 → 平台标准模型名」反查表。
// upstreamModelAlias 构建「上游模型名 → 平台标准模型名」反查表。
// 账号 credentials.model_mapping 的方向是 {平台标准名: 上游原始名}，这里做反向索引。
func (s *UpstreamService) upstreamModelAlias(ctx context.Context, upstreamID int64) (map[string]string, error) {
	accounts, err := s.repo.ListUpstreamAccounts(ctx, upstreamID)
	if err != nil {
		return nil, err
	}
	aliases := make(map[string]string)
	for _, a := range accounts {
		for platformName, upstreamName := range a.ModelMapping {
			key := strings.ToLower(strings.TrimSpace(upstreamName))
			target := strings.TrimSpace(platformName)
			if key == "" || target == "" {
				continue
			}
			aliases[key] = target
		}
	}
	return aliases, nil
}

// applyModelNameFallback 台账 model_name 缺省时按账号映射反查；
// 返回 true 表示成功对齐到平台标准名，false 表示已回落到上游名（调用方据此记录 unmatched）。
func (s *UpstreamService) applyModelNameFallback(ctx context.Context, upstreamID int64, price *UpstreamModelPrice) (bool, error) {
	if strings.TrimSpace(price.ModelName) != "" {
		return true, nil
	}
	aliases, err := s.upstreamModelAlias(ctx, upstreamID)
	if err != nil {
		return false, err
	}
	if mapped := aliases[strings.ToLower(strings.TrimSpace(price.UpstreamModel))]; mapped != "" {
		price.ModelName = mapped
		return true, nil
	}
	price.ModelName = price.UpstreamModel
	return false, nil
}

// resolveGroupID 解析台账归属的上游分组：id 优先，其次按名称查找（不区分大小写），
// 名称不存在时自动创建（导入/爬虫推送通常只带分组名）。
func (s *UpstreamService) resolveGroupID(ctx context.Context, upstreamID int64, groupID *int64, groupName, source string) (*int64, error) {
	if groupID != nil && *groupID > 0 {
		g, err := s.repo.GetUpstreamGroup(ctx, *groupID)
		if err != nil {
			return nil, err
		}
		if g.UpstreamID != upstreamID {
			return nil, ErrUpstreamGroupNotFound
		}
		return groupID, nil
	}
	name := strings.TrimSpace(groupName)
	if name == "" {
		return nil, nil
	}
	groups, err := s.repo.ListUpstreamGroups(ctx, upstreamID)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if strings.EqualFold(groups[i].Name, name) {
			id := groups[i].ID
			return &id, nil
		}
	}
	created, err := s.UpsertGroup(ctx, upstreamID, UpstreamGroupInput{Name: name, Source: source})
	if err != nil {
		return nil, err
	}
	return &created.ID, nil
}

// CompareModels 跨上游比价。
//
// 行 = 平台标准模型名（账号 model_mapping 的 key ∪ 台账 model_name）；
// 列 = status=active 的上游；单元格进价优先级：台账 > 推导（目录价 × 最低账号倍率）。
func (s *UpstreamService) CompareModels(ctx context.Context, platform, search string) (*UpstreamCompareResult, error) {
	upstreams, err := s.repo.ListUpstreams(ctx)
	if err != nil {
		return nil, err
	}
	active := make([]Upstream, 0, len(upstreams))
	for _, u := range upstreams {
		if u.Status == UpstreamStatusActive {
			active = append(active, u)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		return strings.ToLower(active[i].Name) < strings.ToLower(active[j].Name)
	})

	accounts, err := s.repo.ListAllAssignedAccounts(ctx)
	if err != nil {
		return nil, err
	}
	prices, err := s.repo.ListAllUpstreamModelPrices(ctx)
	if err != nil {
		return nil, err
	}

	modelPlatform := make(map[string]string)
	modelAccounts := make(map[string]map[int64][]upstreamAccountRef)
	upstreamHasAccounts := make(map[int64]bool)
	for _, a := range accounts {
		if a.UpstreamID != 0 {
			upstreamHasAccounts[a.UpstreamID] = true
		}
		for source := range a.ModelMapping {
			name := strings.TrimSpace(source)
			if name == "" {
				continue
			}
			if _, ok := modelPlatform[name]; !ok {
				modelPlatform[name] = a.Platform
			}
			if modelAccounts[name] == nil {
				modelAccounts[name] = make(map[int64][]upstreamAccountRef)
			}
			modelAccounts[name][a.UpstreamID] = append(modelAccounts[name][a.UpstreamID], upstreamAccountRef{
				ID:             a.ID,
				Name:           a.Name,
				RateMultiplier: a.RateMultiplier,
			})
		}
	}

	// 台账按 (上游, 分组) 归位；同一 key 有多条时保留进价最低的一条
	ledger := make(map[string]map[int64]map[string]*UpstreamModelPrice)
	for i := range prices {
		p := &prices[i]
		name := strings.TrimSpace(p.ModelName)
		if name == "" {
			name = strings.TrimSpace(p.UpstreamModel)
		}
		if name == "" {
			continue
		}
		if _, ok := modelPlatform[name]; !ok {
			modelPlatform[name] = strings.TrimSpace(p.Platform)
		}
		key := ledgerGroupKey(p.UpstreamGroupID)
		if ledger[name] == nil {
			ledger[name] = make(map[int64]map[string]*UpstreamModelPrice)
		}
		if ledger[name][p.UpstreamID] == nil {
			ledger[name][p.UpstreamID] = make(map[string]*UpstreamModelPrice)
		}
		existing := ledger[name][p.UpstreamID][key]
		if existing == nil || upstreamPriceLess(p, existing) {
			ledger[name][p.UpstreamID][key] = p
		}
	}

	// 列 = 上游 × 分组；上游有账号但未维护分组时补一个「综合」列（账号最低倍率推导）
	columns := make([]UpstreamCompareColumn, 0, len(active)*2)
	for _, u := range active {
		groups, groupErr := s.repo.ListUpstreamGroups(ctx, u.ID)
		if groupErr != nil {
			return nil, groupErr
		}
		sort.Slice(groups, func(i, j int) bool {
			return strings.ToLower(groups[i].Name) < strings.ToLower(groups[j].Name)
		})
		for i := range groups {
			g := groups[i]
			id := g.ID
			columns = append(columns, UpstreamCompareColumn{
				UpstreamID:     u.ID,
				UpstreamName:   u.Name,
				GroupID:        &id,
				GroupName:      g.Name,
				RateMultiplier: g.RateMultiplier,
				Kind:           UpstreamCompareColumnGroup,
			})
		}
		if upstreamHasAccounts[u.ID] {
			columns = append(columns, UpstreamCompareColumn{
				UpstreamID:   u.ID,
				UpstreamName: u.Name,
				GroupName:    UpstreamCompareDerivedColumnName,
				Kind:         UpstreamCompareColumnDerived,
			})
		}
	}

	rows := make([]UpstreamCompareRow, 0, len(modelPlatform))
	searchLower := strings.ToLower(strings.TrimSpace(search))
	for name, rowPlatform := range modelPlatform {
		if platform != "" && !strings.EqualFold(rowPlatform, platform) {
			continue
		}
		if searchLower != "" && !strings.Contains(strings.ToLower(name), searchLower) {
			continue
		}
		row := UpstreamCompareRow{
			ModelName: name,
			Platform:  rowPlatform,
			Cells:     make([]UpstreamCompareCell, 0),
		}
		if s.billingService != nil {
			if mp, err := s.billingService.GetModelPricing(name); err == nil && mp != nil {
				row.OfficialInputPricePerToken = nonZeroFloatPtr(mp.InputPricePerToken)
				row.OfficialOutputPricePerToken = nonZeroFloatPtr(mp.OutputPricePerToken)
			}
		}
		var (
			cheapestUpstreamID *int64
			cheapestGroupID    *int64
			cheapestInput      float64
		)
		for _, col := range columns {
			cell := s.buildCompareCellForColumn(col, name, row.OfficialInputPricePerToken, ledger[name][col.UpstreamID], modelAccounts[name])
			if cell == nil {
				continue
			}
			row.Cells = append(row.Cells, *cell)
			if cell.InputPricePerToken != nil && *cell.InputPricePerToken > 0 {
				if cheapestUpstreamID == nil || *cell.InputPricePerToken < cheapestInput {
					cheapestInput = *cell.InputPricePerToken
					upID := cell.UpstreamID
					cheapestUpstreamID = &upID
					if cell.GroupID != nil {
						gid := *cell.GroupID
						cheapestGroupID = &gid
					}
				}
			}
		}
		if len(row.Cells) >= 2 && cheapestUpstreamID != nil {
			row.CheapestUpstreamID = cheapestUpstreamID
			row.CheapestGroupID = cheapestGroupID
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		return strings.ToLower(rows[i].ModelName) < strings.ToLower(rows[j].ModelName)
	})
	return &UpstreamCompareResult{Upstreams: active, Columns: columns, Rows: rows}, nil
}

// UpstreamCompareDerivedColumnName 综合列（无分组）在表头显示的名字。
const UpstreamCompareDerivedColumnName = "综合"

// upstreamAccountRef 比价推导用的账号引用。
type upstreamAccountRef struct {
	ID             int64
	Name           string
	RateMultiplier float64
}

// ledgerGroupKey 台账按分组归位的索引键。
func ledgerGroupKey(groupID *int64) string {
	if groupID == nil {
		return "-"
	}
	return "g:" + strconv.FormatInt(*groupID, 10)
}

// buildCompareCellForColumn 计算某列（上游×分组）对某模型的进价。
//
// 优先级：该分组台账价 → 分组倍率推导（目录价 × 分组倍率）→ 综合列用账号最低倍率推导。
func (s *UpstreamService) buildCompareCellForColumn(
	col UpstreamCompareColumn,
	modelName string,
	officialInput *float64,
	ledgerForUpstream map[string]*UpstreamModelPrice,
	accountsForUpstream map[int64][]upstreamAccountRef,
) *UpstreamCompareCell {
	cell := &UpstreamCompareCell{
		UpstreamID:   col.UpstreamID,
		GroupID:      col.GroupID,
		UpstreamName: col.UpstreamName,
	}

	// 1. 台账价优先
	if p := ledgerForUpstream[ledgerGroupKey(col.GroupID)]; p != nil {
		multiplier := 1.0
		if p.RateMultiplier != nil && *p.RateMultiplier > 0 {
			multiplier = *p.RateMultiplier
			cell.RateMultiplier = p.RateMultiplier
		}
		cell.InputPricePerToken = scalePricePtr(p.InputPricePerToken, multiplier)
		cell.OutputPricePerToken = scalePricePtr(p.OutputPricePerToken, multiplier)
		cell.Source = upstreamCompareSourceLedger
		if cell.InputPricePerToken != nil || cell.OutputPricePerToken != nil {
			return cell
		}
		// 台账只有按次价（per_request）时不参与 token 比价，回落到推导
	}

	if col.Kind == UpstreamCompareColumnGroup {
		// 2. 分组倍率推导：目录价 × upstream_groups.rate_multiplier
		if col.RateMultiplier != nil && *col.RateMultiplier > 0 && officialInput != nil && *officialInput > 0 {
			cell.RateMultiplier = col.RateMultiplier
			cell.InputPricePerToken = scalePricePtr(officialInput, *col.RateMultiplier)
			cell.Source = upstreamCompareSourceDerived
			return cell
		}
		return nil
	}

	// 3. 综合列：该上游支持该模型的账号中最低倍率推导
	refs := accountsForUpstream[col.UpstreamID]
	if len(refs) == 0 {
		return nil
	}
	lowest := refs[0]
	for _, ref := range refs[1:] {
		if ref.RateMultiplier < lowest.RateMultiplier {
			lowest = ref
		}
	}
	lowestRate := lowest.RateMultiplier
	cell.RateMultiplier = &lowestRate
	cell.Source = upstreamCompareSourceDerived
	accountID := lowest.ID
	cell.AccountID = &accountID
	cell.AccountName = lowest.Name
	cell.InputPricePerToken = scalePricePtr(officialInput, lowest.RateMultiplier)
	if cell.InputPricePerToken == nil {
		return nil
	}
	return cell
}

// upstreamPriceLess a 的输入价是否低于 b（nil 视为无穷大）。
func upstreamPriceLess(a, b *UpstreamModelPrice) bool {
	if a.InputPricePerToken == nil {
		return false
	}
	if b.InputPricePerToken == nil {
		return true
	}
	return *a.InputPricePerToken < *b.InputPricePerToken
}

func scalePricePtr(v *float64, multiplier float64) *float64 {
	if v == nil {
		return nil
	}
	scaled := *v * multiplier
	return &scaled
}

func nonZeroFloatPtr(v float64) *float64 {
	if v == 0 {
		return nil
	}
	return &v
}

// ProvideUpstreamService 构造上游管理服务。
func ProvideUpstreamService(repo UpstreamRepository, billingService *BillingService) *UpstreamService {
	return NewUpstreamService(repo, billingService)
}

func normalizeUpstreamStatus(status string) string {
	if strings.EqualFold(strings.TrimSpace(status), UpstreamStatusInactive) {
		return UpstreamStatusInactive
	}
	return UpstreamStatusActive
}

func validateNonNegative(v *float64, field string) error {
	if v == nil {
		return nil
	}
	if *v < 0 {
		return infraerrors.BadRequest("UPSTREAM_PRICE_INVALID", fmt.Sprintf("%s must be >= 0", field))
	}
	return nil
}
