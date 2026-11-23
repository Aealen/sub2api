package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	// ErrUpstreamBalanceUnavailable 平台余额监控能力不可用（服务未注入）。
	ErrUpstreamBalanceUnavailable = infraerrors.ServiceUnavailable(
		"UPSTREAM_BALANCE_UNAVAILABLE", "upstream balance monitor is unavailable",
	)
)

// 上游余额监控默认值
const (
	// DefaultUpstreamBalanceSyncIntervalSeconds 默认同步间隔（10 分钟）。
	// 30s/1min 属于高频轮询，转售站对 /v1/usage 的限流策略未知，高频有触发风控的风险。
	DefaultUpstreamBalanceSyncIntervalSeconds = 600
	// DefaultUpstreamBalanceAlertThreshold 全局默认预警阈值（美元）。
	DefaultUpstreamBalanceAlertThreshold = 10.0
	// DefaultUpstreamBalanceAlertEmail 默认预警邮箱。
	DefaultUpstreamBalanceAlertEmail = "support@aowu-ai.moe"
	// upstreamUsagePath 上游余额接口路径（勿忘/华龙/PPDog 等转售站统一约定）。
	upstreamUsagePath = "/v1/usage"
	// upstreamUsageTimeout 单账号余额拉取超时。
	upstreamUsageTimeout = 15 * time.Second
	// upstreamUsageConcurrency 单轮同步的并发上限。
	upstreamUsageConcurrency = 4
	// upstreamBalanceLeaderLockKey 多实例定时同步领导者锁键。
	upstreamBalanceLeaderLockKey = "upstream:balance:sync:leader"
	// upstreamBalanceLeaderLockTTL 领导者锁 TTL（覆盖一个同步周期即可）。
	upstreamBalanceLeaderLockTTL = 2 * time.Minute
	// 同步滴答周期：定时器固定 30s 滴答，按配置间隔判断是否到点，间隔改动实时生效。
	balanceSyncTickInterval = 30 * time.Second
	// extra 键：告警防轰炸标记（跌破只发一次，回升重置）
	extraKeyBalanceAlertActive = "balance_alert_active"
	// extra 键：上次告警时间（RFC3339）
	extraKeyBalanceAlertLastAt = "balance_alert_last_at"
)

// UpstreamBalanceAccountStore 上游余额服务需要的账号存取能力（消费者定义接口）。
type UpstreamBalanceAccountStore interface {
	GetByID(ctx context.Context, id int64) (*Account, error)
	ListActive(ctx context.Context) ([]Account, error)
	Update(ctx context.Context, account *Account) error
	// MergeAccountExtra 将 patch 合并进账号 extra JSON（JSONB || 语义，不覆盖其他键）。
	MergeAccountExtra(ctx context.Context, id int64, patch map[string]any) error
	// UpdateUpstreamBalance 仅更新余额与同步时间两列，避免全量覆盖的并发覆盖风险。
	UpdateUpstreamBalance(ctx context.Context, id int64, balance float64, at time.Time) error
	// UpdateBalanceAlertThreshold 更新账号专属预警阈值；threshold 为 nil 表示清除（回落全局默认）。
	UpdateBalanceAlertThreshold(ctx context.Context, id int64, threshold *float64) error
	// ListAccountsWithAlertThreshold 列出设置了专属预警阈值的账号（全量，跨分页）。
	ListAccountsWithAlertThreshold(ctx context.Context) ([]Account, error)
}

// UpstreamBalanceService 定期拉取转售站账号的上游余额（GET {base_url}/v1/usage），
// 在账号列表展示，并在余额跌破预警阈值时给管理员邮箱发预警邮件。
//
// 防轰炸设计：余额跌破阈值只发一次邮件并置 extra.balance_alert_active 标记；
// 持续低于阈值不再重复发送；余额回升到阈值以上时清除标记，下次跌破可重新告警。
type UpstreamBalanceService struct {
	accountRepo  UpstreamBalanceAccountStore
	settingRepo  SettingRepository
	emailService *EmailService
	siteName     func(ctx context.Context) string
	// lockCache 多实例部署时的定时同步领导者锁（单实例/手动触发不受影响）；nil 则不锁。
	lockCache LeaderLockCache
	owner     string

	stopCh     chan struct{}
	stopOnce   sync.Once
	wg         sync.WaitGroup
	syncMu     sync.Mutex // 保护 lastSyncAt（定时器滴答与手动触发并发）
	lastSyncAt time.Time
}

// SetLeaderLock 注入多实例领导者锁（蓝绿/多容器部署时避免双实例同时跑定时同步）。
func (s *UpstreamBalanceService) SetLeaderLock(lockCache LeaderLockCache) {
	if s == nil {
		return
	}
	s.lockCache = lockCache
	s.owner = fmt.Sprintf("upstream-balance:%d", time.Now().UnixNano())
}

// UpstreamBalanceUsageResponse 上游 /v1/usage 响应中本服务关心的字段。
type UpstreamBalanceUsageResponse struct {
	Balance  *float64       `json:"balance"`
	PlanName string         `json:"planName"`
	Unit     string         `json:"unit"`
	Usage    map[string]any `json:"usage"`
}

// UpstreamBalanceConfig 上游余额监控的全局配置。
type UpstreamBalanceConfig struct {
	SyncIntervalSeconds int      `json:"sync_interval_seconds"`
	AlertEnabled        bool     `json:"alert_enabled"`
	AlertEmails         []string `json:"alert_emails"`
	AlertThreshold      float64  `json:"alert_threshold"`
}

func NewUpstreamBalanceService(
	accountRepo UpstreamBalanceAccountStore,
	settingRepo SettingRepository,
	emailService *EmailService,
	siteName func(ctx context.Context) string,
) *UpstreamBalanceService {
	return &UpstreamBalanceService{
		accountRepo:  accountRepo,
		settingRepo:  settingRepo,
		emailService: emailService,
		siteName:     siteName,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动同步定时器。滴答固定 30s，每滴答读一次配置间隔判断是否到点，
// 间隔配置改动无需重启即可生效。
func (s *UpstreamBalanceService) Start() {
	if s == nil || s.accountRepo == nil {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		baseCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ticker := time.NewTicker(balanceSyncTickInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				cfg, err := s.LoadConfig(baseCtx)
				if err != nil {
					slog.Warn("upstream balance: load sync config failed; skip tick", "err", err.Error())
					continue
				}
				if cfg.SyncIntervalSeconds <= 0 {
					continue
				}
				s.syncMu.Lock()
				due := time.Since(s.lastSyncAt) >= time.Duration(cfg.SyncIntervalSeconds)*time.Second
				if due {
					s.lastSyncAt = time.Now()
				}
				s.syncMu.Unlock()
				if !due {
					continue
				}
				// 多实例领导者锁：拿不到锁（他人持有）时跳过本轮定时同步。
				if s.lockCache != nil {
					ok, err := s.lockCache.TryAcquireLeaderLock(baseCtx, upstreamBalanceLeaderLockKey, s.owner, upstreamBalanceLeaderLockTTL)
					if err != nil {
						slog.Warn("upstream balance: acquire leader lock failed; run anyway", "err", err.Error())
					} else if !ok {
						continue
					}
				}
				s.SyncAll(baseCtx)
			case <-s.stopCh:
				return
			}
		}
	}()
}

// Stop 停止定时器并等待在途同步结束。
func (s *UpstreamBalanceService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

// LoadConfig 读取全局配置并应用默认值。
func (s *UpstreamBalanceService) LoadConfig(ctx context.Context) (UpstreamBalanceConfig, error) {
	settings, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyUpstreamBalanceSyncIntervalSeconds,
		SettingKeyUpstreamBalanceAlertEnabled,
		SettingKeyUpstreamBalanceAlertEmails,
		SettingKeyUpstreamBalanceAlertThreshold,
	})
	if err != nil {
		return UpstreamBalanceConfig{}, err
	}

	cfg := UpstreamBalanceConfig{
		SyncIntervalSeconds: DefaultUpstreamBalanceSyncIntervalSeconds,
		AlertEnabled:        true,
		AlertThreshold:      DefaultUpstreamBalanceAlertThreshold,
		AlertEmails:         []string{DefaultUpstreamBalanceAlertEmail},
	}
	if v := strings.TrimSpace(settings[SettingKeyUpstreamBalanceSyncIntervalSeconds]); v != "" {
		if n := parseIntDefault(v, cfg.SyncIntervalSeconds); n > 0 {
			cfg.SyncIntervalSeconds = n
		}
	}
	if settings[SettingKeyUpstreamBalanceAlertEnabled] == "false" {
		cfg.AlertEnabled = false
	}
	if raw := strings.TrimSpace(settings[SettingKeyUpstreamBalanceAlertEmails]); raw != "" && raw != "[]" {
		if emails := notifyEmailEntriesToAddresses(ParseNotifyEmails(raw)); len(emails) > 0 {
			cfg.AlertEmails = emails
		}
	}
	if v := strings.TrimSpace(settings[SettingKeyUpstreamBalanceAlertThreshold]); v != "" {
		if f := parseFloatDefault(v, cfg.AlertThreshold); f >= 0 {
			cfg.AlertThreshold = f
		}
	}
	return cfg, nil
}

// SaveConfig 保存全局配置（预警配置弹窗保存）。
func (s *UpstreamBalanceService) SaveConfig(ctx context.Context, cfg UpstreamBalanceConfig) error {
	if cfg.SyncIntervalSeconds <= 0 {
		return fmt.Errorf("sync_interval_seconds must be positive")
	}
	if cfg.AlertThreshold < 0 {
		return fmt.Errorf("alert_threshold must be >= 0")
	}
	emails := make([]string, 0, len(cfg.AlertEmails))
	for _, e := range cfg.AlertEmails {
		if addr := strings.TrimSpace(e); addr != "" {
			emails = append(emails, addr)
		}
	}
	rawEmails, err := json.Marshal(emails)
	if err != nil {
		return err
	}
	return s.settingRepo.SetMultiple(ctx, map[string]string{
		SettingKeyUpstreamBalanceSyncIntervalSeconds: strconv.Itoa(cfg.SyncIntervalSeconds),
		SettingKeyUpstreamBalanceAlertEnabled:        strconv.FormatBool(cfg.AlertEnabled),
		SettingKeyUpstreamBalanceAlertEmails:         string(rawEmails),
		SettingKeyUpstreamBalanceAlertThreshold:      strconv.FormatFloat(cfg.AlertThreshold, 'f', -1, 64),
	})
}

// SetAccountAlertThreshold 设置账号专属预警阈值（覆盖全局默认）；threshold 为 nil 清除。
func (s *UpstreamBalanceService) SetAccountAlertThreshold(ctx context.Context, id int64, threshold *float64) error {
	if _, err := s.accountRepo.GetByID(ctx, id); err != nil {
		return err
	}
	return s.accountRepo.UpdateBalanceAlertThreshold(ctx, id, threshold)
}

// SyncAll 拉取全部可同步账号的上游余额并做阈值检查。
// 可同步账号 = active + apikey 型 + credentials 带 base_url 与 api_key。
func (s *UpstreamBalanceService) SyncAll(ctx context.Context) {
	s.syncMu.Lock()
	s.lastSyncAt = time.Now()
	s.syncMu.Unlock()

	accounts, err := s.accountRepo.ListActive(ctx)
	if err != nil {
		slog.Warn("upstream balance: list accounts failed", "err", err.Error())
		return
	}
	targets := make([]*Account, 0, len(accounts))
	for i := range accounts {
		a := &accounts[i]
		if a.Type != AccountTypeAPIKey || a.GetBaseURL() == "" || a.GetCredential("api_key") == "" {
			continue
		}
		targets = append(targets, a)
	}
	if len(targets) == 0 {
		return
	}

	cfg, err := s.LoadConfig(ctx)
	if err != nil {
		slog.Warn("upstream balance: load config failed; skip alert check", "err", err.Error())
	}

	sem := make(chan struct{}, upstreamUsageConcurrency)
	var wg sync.WaitGroup
	for _, acc := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(a *Account) {
			defer wg.Done()
			defer func() { <-sem }()
			s.syncOne(ctx, a, cfg)
		}(acc)
	}
	wg.Wait()
}

// SyncAccount 立即拉取单个账号的上游余额（手动刷新按钮）。返回刷新后的余额。
func (s *UpstreamBalanceService) SyncAccount(ctx context.Context, id int64) (*float64, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if account.Type != AccountTypeAPIKey || account.GetBaseURL() == "" || account.GetCredential("api_key") == "" {
		return nil, fmt.Errorf("account %d is not an apikey account with base_url/api_key", id)
	}
	cfg, cfgErr := s.LoadConfig(ctx)
	balance, err := s.fetchBalance(ctx, account.GetBaseURL(), account.GetCredential("api_key"))
	if err != nil {
		return nil, err
	}
	if err := s.accountRepo.UpdateUpstreamBalance(ctx, id, balance, time.Now()); err != nil {
		return nil, err
	}
	if cfgErr == nil {
		s.checkAlert(ctx, account, balance, cfg)
	}
	return &balance, nil
}

// syncOne 拉取单账号余额、落库并做阈值检查。失败静默（warn 日志），不影响其他账号。
func (s *UpstreamBalanceService) syncOne(ctx context.Context, account *Account, cfg UpstreamBalanceConfig) {
	balance, err := s.fetchBalance(ctx, account.GetBaseURL(), account.GetCredential("api_key"))
	if err != nil {
		slog.Warn("upstream balance: fetch failed; keep previous value",
			"account_id", account.ID, "account_name", account.Name, "err", err.Error())
		return
	}
	if err := s.accountRepo.UpdateUpstreamBalance(ctx, account.ID, balance, time.Now()); err != nil {
		slog.Warn("upstream balance: persist failed", "account_id", account.ID, "err", err.Error())
		return
	}
	s.checkAlert(ctx, account, balance, cfg)
}

// checkAlert 阈值检查与预警邮件（防轰炸：跌破只发一次，回升重置）。
func (s *UpstreamBalanceService) checkAlert(ctx context.Context, account *Account, balance float64, cfg UpstreamBalanceConfig) {
	if !cfg.AlertEnabled || len(cfg.AlertEmails) == 0 {
		return
	}
	threshold := cfg.AlertThreshold
	if account.BalanceAlertThreshold != nil && *account.BalanceAlertThreshold > 0 {
		threshold = *account.BalanceAlertThreshold
	}

	active := accountExtraBool(account.Extra, extraKeyBalanceAlertActive)
	if balance < threshold && !active {
		// 跌破：先置标记再发邮件，避免并发重复告警；发信失败也保留标记（下轮不再轰炸，可手动重置）。
		if err := s.accountRepo.MergeAccountExtra(ctx, account.ID, map[string]any{
			extraKeyBalanceAlertActive: true,
			extraKeyBalanceAlertLastAt: time.Now().Format(time.RFC3339),
		}); err != nil {
			slog.Warn("upstream balance: mark alert state failed", "account_id", account.ID, "err", err.Error())
			return
		}
		s.sendAlertEmails(ctx, cfg.AlertEmails, account, balance, threshold)
		slog.Info("upstream balance: alert dispatched",
			"account_id", account.ID, "account_name", account.Name, "balance", balance, "threshold", threshold)
		return
	}
	if balance >= threshold && active {
		// 回升：清除标记，下次跌破可重新告警。
		if err := s.accountRepo.MergeAccountExtra(ctx, account.ID, map[string]any{
			extraKeyBalanceAlertActive: false,
		}); err != nil {
			slog.Warn("upstream balance: clear alert state failed", "account_id", account.ID, "err", err.Error())
			return
		}
		slog.Info("upstream balance: alert recovered; state reset",
			"account_id", account.ID, "account_name", account.Name, "balance", balance, "threshold", threshold)
	}
}

// sendAlertEmails 给全部预警邮箱发送余额预警邮件（单封失败不阻断其他收件人）。
func (s *UpstreamBalanceService) sendAlertEmails(ctx context.Context, emails []string, account *Account, balance, threshold float64) {
	if s.emailService == nil {
		return
	}
	siteName := "Sub2API"
	if s.siteName != nil {
		if v := strings.TrimSpace(s.siteName(ctx)); v != "" {
			siteName = v
		}
	}
	subject := fmt.Sprintf("【%s】上游余额预警：%s 余额 $%.2f，低于阈值 $%.2f", siteName, account.Name, balance, threshold)
	body := fmt.Sprintf(`<div style="font-family:Arial,'Microsoft YaHei',sans-serif;max-width:560px;margin:0 auto;">
<h2 style="color:#c0392b;">上游余额预警</h2>
<p>监控的上游账号余额已跌破预警阈值，请及时关注并充值，避免影响该账号承载的模型服务。</p>
<table style="border-collapse:collapse;width:100%%;" cellpadding="8">
<tr><td style="border:1px solid #ddd;background:#f7f7f7;">账号名称</td><td style="border:1px solid #ddd;">%s（ID: %d）</td></tr>
<tr><td style="border:1px solid #ddd;background:#f7f7f7;">平台 / 类型</td><td style="border:1px solid #ddd;">%s / %s</td></tr>
<tr><td style="border:1px solid #ddd;background:#f7f7f7;">当前余额</td><td style="border:1px solid #ddd;color:#c0392b;font-weight:bold;">$%.2f</td></tr>
<tr><td style="border:1px solid #ddd;background:#f7f7f7;">预警阈值</td><td style="border:1px solid #ddd;">$%.2f</td></tr>
<tr><td style="border:1px solid #ddd;background:#f7f7f7;">告警时间</td><td style="border:1px solid #ddd;">%s</td></tr>
</table>
<p style="color:#888;font-size:12px;">本邮件由上游余额监控自动发送；余额回升到阈值以上后将自动重置，再次跌破会重新告警。</p>
</div>`, account.Name, account.ID, account.Platform, account.Type,
		balance, threshold, time.Now().Format("2006-01-02 15:04:05"))

	for _, to := range emails {
		sendCtx, cancel := context.WithTimeout(ctx, emailSendTimeout)
		err := s.emailService.SendEmail(sendCtx, to, subject, body)
		cancel()
		if err != nil {
			slog.Warn("upstream balance: alert email delivery failed", "to", to, "account_id", account.ID, "err", err.Error())
		}
	}
}

// fetchBalance 调上游 GET {base_url}/v1/usage 取 balance。
func (s *UpstreamBalanceService) fetchBalance(ctx context.Context, baseURL, apiKey string) (float64, error) {
	url := strings.TrimRight(baseURL, "/") + upstreamUsagePath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: upstreamUsageTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("upstream usage api returned %d", resp.StatusCode)
	}
	var parsed UpstreamBalanceUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return 0, fmt.Errorf("decode usage response: %w", err)
	}
	if parsed.Balance == nil {
		return 0, fmt.Errorf("usage response has no balance field")
	}
	return *parsed.Balance, nil
}

// notifyEmailEntriesToAddresses 从邮箱条目中取启用且已验证的地址列表。
func notifyEmailEntriesToAddresses(entries []NotifyEmailEntry) []string {
	addresses := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Disabled {
			continue
		}
		addr := strings.TrimSpace(e.Email)
		if addr != "" {
			addresses = append(addresses, addr)
		}
	}
	return addresses
}

func accountExtraBool(extra map[string]any, key string) bool {
	if extra == nil {
		return false
	}
	v, ok := extra[key].(bool)
	return ok && v
}

func parseIntDefault(v string, fallback int) int {
	var n int
	if _, err := fmt.Sscanf(strings.TrimSpace(v), "%d", &n); err != nil || n <= 0 {
		return fallback
	}
	return n
}

func parseFloatDefault(v string, fallback float64) float64 {
	var f float64
	if _, err := fmt.Sscanf(strings.TrimSpace(v), "%g", &f); err != nil {
		return fallback
	}
	return f
}

// ProvideUpstreamBalanceService wire 注入构造器。
func ProvideUpstreamBalanceService(
	accountRepo AccountRepository,
	settingRepo SettingRepository,
	emailService *EmailService,
	lockCache LeaderLockCache,
) *UpstreamBalanceService {
	store, ok := any(accountRepo).(UpstreamBalanceAccountStore)
	if !ok || store == nil {
		slog.Warn("upstream balance: account repository lacks balance store methods; service disabled")
		return nil
	}
	siteName := func(ctx context.Context) string {
		if v, err := settingRepo.GetValue(ctx, SettingKeySiteName); err == nil {
			return v
		}
		return ""
	}
	svc := NewUpstreamBalanceService(store, settingRepo, emailService, siteName)
	svc.SetLeaderLock(lockCache)
	svc.Start()
	return svc
}

// UpstreamBalanceThresholdOverride 已设置专属预警阈值的账号（配置弹窗展示用）。
type UpstreamBalanceThresholdOverride struct {
	AccountID   int64   `json:"account_id"`
	AccountName string  `json:"account_name"`
	Threshold   float64 `json:"threshold"`
}

// ListThresholdOverrides 列出全部专属预警阈值覆盖。
func (s *UpstreamBalanceService) ListThresholdOverrides(ctx context.Context) ([]UpstreamBalanceThresholdOverride, error) {
	accounts, err := s.accountRepo.ListAccountsWithAlertThreshold(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]UpstreamBalanceThresholdOverride, 0, len(accounts))
	for i := range accounts {
		a := &accounts[i]
		if a.BalanceAlertThreshold == nil {
			continue
		}
		out = append(out, UpstreamBalanceThresholdOverride{
			AccountID:   a.ID,
			AccountName: a.Name,
			Threshold:   *a.BalanceAlertThreshold,
		})
	}
	return out, nil
}
