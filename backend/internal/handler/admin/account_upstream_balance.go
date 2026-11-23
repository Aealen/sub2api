package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// upstreamBalanceRefreshRequest 手动刷新请求：account_id 缺省时全量刷新。
type upstreamBalanceRefreshRequest struct {
	AccountID *int64 `json:"account_id"`
}

// GetUpstreamBalanceConfig 返回预警配置弹窗所需的全部数据：
// 全局配置、已设置的账号专属阈值、可同步余额的全部账号（账号下拉用）。
func (h *AccountHandler) GetUpstreamBalanceConfig(c *gin.Context) {
	if h.upstreamBalance == nil {
		response.ErrorFrom(c, service.ErrUpstreamBalanceUnavailable)
		return
	}
	cfg, err := h.upstreamBalance.LoadConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	overrides, err := h.upstreamBalance.ListThresholdOverrides(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	accounts, err := h.upstreamBalance.ListEligibleAccounts(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"config":                 cfg,
		"per_account_thresholds": overrides,
		"accounts":               accounts,
	})
}

// UpdateUpstreamBalanceConfig 保存上游余额监控全局配置。
func (h *AccountHandler) UpdateUpstreamBalanceConfig(c *gin.Context) {
	if h.upstreamBalance == nil {
		response.ErrorFrom(c, service.ErrUpstreamBalanceUnavailable)
		return
	}
	var req service.UpstreamBalanceConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.upstreamBalance.SaveConfig(c.Request.Context(), req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	cfg, err := h.upstreamBalance.LoadConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, cfg)
}

// RefreshUpstreamBalance 手动刷新上游余额：带 account_id 刷新单账号，缺省刷新全部。
func (h *AccountHandler) RefreshUpstreamBalance(c *gin.Context) {
	if h.upstreamBalance == nil {
		response.ErrorFrom(c, service.ErrUpstreamBalanceUnavailable)
		return
	}
	var req upstreamBalanceRefreshRequest
	_ = c.ShouldBindJSON(&req) // body 可省略（全量刷新）

	if req.AccountID != nil && *req.AccountID > 0 {
		balance, err := h.upstreamBalance.SyncAccount(c.Request.Context(), *req.AccountID)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, gin.H{"account_id": *req.AccountID, "balance": balance})
		return
	}

	h.upstreamBalance.SyncAll(c.Request.Context())
	response.Success(c, gin.H{"triggered": "all"})
}

// UpdateAccountBalanceAlertThreshold 设置账号专属预警阈值（预警配置弹窗内）。
// threshold 为 null 清除专属设置（回落全局默认）。
func (h *AccountHandler) UpdateAccountBalanceAlertThreshold(c *gin.Context) {
	if h.upstreamBalance == nil {
		response.ErrorFrom(c, service.ErrUpstreamBalanceUnavailable)
		return
	}
	accountID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || accountID <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}
	var req struct {
		Threshold *float64 `json:"threshold"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if req.Threshold != nil && *req.Threshold < 0 {
		response.BadRequest(c, "threshold must be >= 0")
		return
	}
	if err := h.upstreamBalance.SetAccountAlertThreshold(c.Request.Context(), accountID, req.Threshold); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"account_id": accountID, "balance_alert_threshold": req.Threshold})
}
