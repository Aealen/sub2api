/**
 * 定价覆盖文件管理（官方价目录的人民币口径中枢）
 * 后端：GET/PUT /api/v1/admin/channels/pricing-override
 * 写入后服务端立即触发热加载；热加载失败以 warning 语义返回（文件已落盘，周期检测兜底）
 */
import apiClient from '../client'

export interface PricingOverrideResponse {
  /** 是否配置了 override 文件路径（false 时前端提示不可用） */
  configured: boolean
  /** 文件路径（容器内视角） */
  path: string
  /** 当前文件内容（未创建时为空串） */
  content: string
}

export interface UpdatePricingOverrideResult {
  saved: boolean
  reloaded: boolean
  warning?: string
}

export async function getPricingOverride(): Promise<PricingOverrideResponse> {
  const { data } = await apiClient.get<PricingOverrideResponse>('/admin/channels/pricing-override')
  return data
}

export async function updatePricingOverride(content: string): Promise<UpdatePricingOverrideResult> {
  const { data } = await apiClient.put<UpdatePricingOverrideResult>('/admin/channels/pricing-override', { content })
  return data
}

export const pricingOverrideAPI = { getPricingOverride, updatePricingOverride }
export default pricingOverrideAPI
