package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// UpstreamHandler 上游管理（上游库、上游侧分组、模型价格台账、跨上游比价）。
type UpstreamHandler struct {
	upstreamService *service.UpstreamService
}

// NewUpstreamHandler 创建上游管理 handler。
func NewUpstreamHandler(upstreamService *service.UpstreamService) *UpstreamHandler {
	return &UpstreamHandler{upstreamService: upstreamService}
}

func (h *UpstreamHandler) available(c *gin.Context) bool {
	if h == nil || h.upstreamService == nil {
		response.Error(c, http.StatusServiceUnavailable, "upstream service unavailable")
		return false
	}
	return true
}

// List 上游列表（含余额、账号数、倍率区间聚合）。
func (h *UpstreamHandler) List(c *gin.Context) {
	if !h.available(c) {
		return
	}
	items, err := h.upstreamService.ListOverview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": items, "total": len(items)})
}

// Overview 上游经营概览：上游卡片、供货关联、未关联分组与分组流水（7/30 天窗口）。
func (h *UpstreamHandler) Overview(c *gin.Context) {
	if !h.available(c) {
		return
	}
	result, err := h.upstreamService.GetOverview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// Create 新建上游。
func (h *UpstreamHandler) Create(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var req service.UpstreamInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	created, err := h.upstreamService.CreateUpstream(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, created)
}

// GetByID 上游详情（基本信息 + 账号 + 上游分组 + 台账）。
func (h *UpstreamHandler) GetByID(c *gin.Context) {
	if !h.available(c) {
		return
	}
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	detail, err := h.upstreamService.GetDetail(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, detail)
}

// Update 更新上游元信息。
func (h *UpstreamHandler) Update(c *gin.Context) {
	if !h.available(c) {
		return
	}
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	var req service.UpstreamInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	updated, err := h.upstreamService.UpdateUpstream(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}

// Delete 软删上游。
func (h *UpstreamHandler) Delete(c *gin.Context) {
	if !h.available(c) {
		return
	}
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	if err := h.upstreamService.DeleteUpstream(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ListUnassignedAccounts 未归属上游的账号，按 base_url host 聚合。
func (h *UpstreamHandler) ListUnassignedAccounts(c *gin.Context) {
	if !h.available(c) {
		return
	}
	hosts, err := h.upstreamService.ListUnassignedAccountHosts(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": hosts})
}

type assignUpstreamAccountsRequest struct {
	AccountIDs []int64 `json:"account_ids"`
}

// AssignAccounts 批量把账号挂到该上游。
func (h *UpstreamHandler) AssignAccounts(c *gin.Context) {
	if !h.available(c) {
		return
	}
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	var req assignUpstreamAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	if err := h.upstreamService.AssignAccounts(c.Request.Context(), id, req.AccountIDs); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"assigned": len(req.AccountIDs)})
}

// ListGroups 上游侧分组列表。
func (h *UpstreamHandler) ListGroups(c *gin.Context) {
	if !h.available(c) {
		return
	}
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	detail, err := h.upstreamService.GetDetail(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": detail.Groups})
}

// UpsertGroup 新建或更新上游侧分组。
func (h *UpstreamHandler) UpsertGroup(c *gin.Context) {
	if !h.available(c) {
		return
	}
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	var req service.UpstreamGroupInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	group, err := h.upstreamService.UpsertGroup(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, group)
}

// DeleteGroup 删除上游侧分组。
func (h *UpstreamHandler) DeleteGroup(c *gin.Context) {
	if !h.available(c) {
		return
	}
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	groupID, err := strconv.ParseInt(c.Param("groupId"), 10, 64)
	if err != nil || groupID <= 0 {
		response.BadRequest(c, "invalid group id")
		return
	}
	if err := h.upstreamService.DeleteGroup(c.Request.Context(), id, groupID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ListGroupLinks 本站分组 ↔ 上游分组的供货关联列表。
func (h *UpstreamHandler) ListGroupLinks(c *gin.Context) {
	if !h.available(c) {
		return
	}
	links, err := h.upstreamService.ListGroupLinks(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": links})
}

// UpsertGroupLink 建立或更新供货关联（同一对分组幂等覆盖）。
func (h *UpstreamHandler) UpsertGroupLink(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var req service.GroupUpstreamLinkInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	link, err := h.upstreamService.UpsertGroupLink(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, link)
}

// DeleteGroupLink 软删供货关联。
func (h *UpstreamHandler) DeleteGroupLink(c *gin.Context) {
	if !h.available(c) {
		return
	}
	linkID, err := strconv.ParseInt(c.Param("linkId"), 10, 64)
	if err != nil || linkID <= 0 {
		response.BadRequest(c, "invalid link id")
		return
	}
	if err := h.upstreamService.DeleteGroupLink(c.Request.Context(), linkID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// ListModelPrices 模型价格台账列表。
func (h *UpstreamHandler) ListModelPrices(c *gin.Context) {
	if !h.available(c) {
		return
	}
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	detail, err := h.upstreamService.GetDetail(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": detail.ModelPrices})
}

// UpsertModelPrice 单条台账写入。
func (h *UpstreamHandler) UpsertModelPrice(c *gin.Context) {
	if !h.available(c) {
		return
	}
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	var req service.UpstreamModelPriceInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	price, err := h.upstreamService.UpsertModelPrice(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, price)
}

// DeleteModelPrice 删除台账条目。
func (h *UpstreamHandler) DeleteModelPrice(c *gin.Context) {
	if !h.available(c) {
		return
	}
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	priceID, err := strconv.ParseInt(c.Param("priceId"), 10, 64)
	if err != nil || priceID <= 0 {
		response.BadRequest(c, "invalid price id")
		return
	}
	if err := h.upstreamService.DeleteModelPrice(c.Request.Context(), id, priceID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

type importUpstreamModelPricesRequest struct {
	Source string                             `json:"source"`
	Items  []service.UpstreamModelPriceInput `json:"items"`
}

// ImportModelPrices 批量导入台账（幂等 upsert）；source=import/crawler。
func (h *UpstreamHandler) ImportModelPrices(c *gin.Context) {
	if !h.available(c) {
		return
	}
	id, ok := parseUpstreamID(c)
	if !ok {
		return
	}
	var req importUpstreamModelPricesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	if len(req.Items) == 0 {
		response.BadRequest(c, "items is required")
		return
	}
	result, err := h.upstreamService.ImportModelPrices(c.Request.Context(), id, req.Items, req.Source)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

// ImportBundle 上游整包导入（供外部爬虫/脚本一次推送一家上游的完整配置）。
func (h *UpstreamHandler) ImportBundle(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var req service.UpstreamBundleImport
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	source := strings.TrimSpace(c.Query("source"))
	upstream, result, err := h.upstreamService.ImportBundle(c.Request.Context(), req, source)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"upstream": upstream, "import": result})
}

// Compare 跨上游比价矩阵。
func (h *UpstreamHandler) Compare(c *gin.Context) {
	if !h.available(c) {
		return
	}
	platform := strings.TrimSpace(c.Query("platform"))
	search := strings.TrimSpace(c.Query("search"))
	if len(search) > 100 {
		search = search[:100]
	}
	result, err := h.upstreamService.CompareModels(c.Request.Context(), platform, search)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func parseUpstreamID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid upstream id")
		return 0, false
	}
	return id, true
}
