import { apiClient } from '../client'

/**
 * 上游管理 API。
 *
 * 价格字段统一以 USD / token 传输（与后端计费口径一致），
 * 页面展示时 ×1e6 转成「每百万 token」。
 */
export interface Upstream {
  id: number
  name: string
  base_url: string
  website: string
  contact: string
  note: string
  status: 'active' | 'inactive'
  created_at: string
  updated_at: string
}

export interface UpstreamOverview extends Upstream {
  account_total: number
  account_active: number
  site_balance: number
  balance_updated_at: string | null
  rate_multiplier_min: number | null
  rate_multiplier_max: number | null
  model_price_count: number
  group_count: number
  group_names: string[]
}

export interface UpstreamAccountRow {
  id: number
  upstream_id: number
  name: string
  platform: string
  type: string
  status: string
  base_url: string
  rate_multiplier: number
  upstream_balance: number | null
  upstream_balance_updated_at: string | null
  model_mapping: Record<string, string>
  mapped_model_count: number
  group_ids: number[]
  group_names: string[]
  probed_group_rate_multiplier: number | null
  probed_effective_rate_multiplier: number | null
  probed_peak_multiplier: number | null
  probed_peak_start: string
  probed_peak_end: string
  probed_observed_at: string
}

export interface UpstreamGroup {
  id: number
  upstream_id: number
  name: string
  rate_multiplier: number | null
  peak_rate_multiplier: number | null
  peak_start: string
  peak_end: string
  source: 'manual' | 'probe' | 'import' | 'crawler'
  note: string
  created_at: string
  updated_at: string
}

export interface UpstreamModelPrice {
  id: number
  upstream_id: number
  upstream_group_id: number | null
  upstream_model: string
  model_name: string
  platform: string
  billing_mode: 'token' | 'per_request' | 'image' | 'video'
  currency: string
  input_price_per_token: number | null
  output_price_per_token: number | null
  cache_read_price_per_token: number | null
  cache_write_price_per_token: number | null
  per_request_price: number | null
  rate_multiplier: number | null
  source: 'manual' | 'import' | 'crawler'
  source_url: string
  effective_at: string | null
  note: string
  created_at: string
  updated_at: string
}

export interface UpstreamDetail extends Upstream {
  accounts: UpstreamAccountRow[]
  groups: UpstreamGroup[]
  model_prices: UpstreamModelPrice[]
}

export interface UpstreamCompareColumn {
  upstream_id: number
  upstream_name: string
  group_id: number | null
  group_name: string
  rate_multiplier: number | null
  kind: 'group' | 'derived'
}

export interface UpstreamCompareCell {
  upstream_id: number
  group_id: number | null
  upstream_name: string
  input_price_per_token: number | null
  output_price_per_token: number | null
  rate_multiplier: number | null
  source: 'ledger' | 'derived'
  account_id?: number
  account_name?: string
}

export interface UpstreamCompareRow {
  model_name: string
  platform: string
  official_input_price_per_token: number | null
  official_output_price_per_token: number | null
  cells: UpstreamCompareCell[]
  cheapest_upstream_id: number | null
  cheapest_group_id: number | null
}

export interface UpstreamCompareResult {
  upstreams: Upstream[]
  columns: UpstreamCompareColumn[]
  rows: UpstreamCompareRow[]
}

export interface UnassignedAccountHost {
  base_url: string
  host: string
  account_count: number
  sample_names: string[]
  account_ids: number[]
}

export interface UpstreamImportResult {
  upserted: number
  skipped: number
  unmatched: string[]
}

export interface UpstreamInput {
  name: string
  base_url?: string
  website?: string
  contact?: string
  note?: string
  status?: 'active' | 'inactive'
}

export interface UpstreamGroupInput {
  id?: number
  name: string
  rate_multiplier?: number | null
  peak_rate_multiplier?: number | null
  peak_start?: string
  peak_end?: string
  source?: UpstreamGroup['source']
  note?: string
}

export interface UpstreamModelPriceInput {
  id?: number
  upstream_group_id?: number | null
  upstream_group_name?: string
  upstream_model: string
  model_name?: string
  platform?: string
  billing_mode?: UpstreamModelPrice['billing_mode']
  currency?: string
  input_price_per_token?: number | null
  output_price_per_token?: number | null
  cache_read_price_per_token?: number | null
  cache_write_price_per_token?: number | null
  per_request_price?: number | null
  rate_multiplier?: number | null
  source?: UpstreamModelPrice['source']
  source_url?: string
  effective_at?: string | null
  note?: string
}

export interface UpstreamBundleImport {
  upstream: UpstreamInput
  groups?: UpstreamGroupInput[]
  model_prices?: UpstreamModelPriceInput[]
}

/** 上游列表（含余额/账号数/倍率区间聚合） */
export async function list(): Promise<{ items: UpstreamOverview[]; total: number }> {
  const { data } = await apiClient.get<{ items: UpstreamOverview[]; total: number }>('/admin/upstreams')
  return data
}

/** 新建上游 */
export async function create(input: UpstreamInput): Promise<Upstream> {
  const { data } = await apiClient.post<Upstream>('/admin/upstreams', input)
  return data
}

/** 上游详情 */
export async function getById(id: number): Promise<UpstreamDetail> {
  const { data } = await apiClient.get<UpstreamDetail>(`/admin/upstreams/${id}`)
  return data
}

/** 更新上游元信息 */
export async function update(id: number, input: UpstreamInput): Promise<Upstream> {
  const { data } = await apiClient.put<Upstream>(`/admin/upstreams/${id}`, input)
  return data
}

/** 软删上游 */
export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/upstreams/${id}`)
}

/** 未归属上游的账号（按 host 聚合） */
export async function unassignedAccounts(): Promise<{ items: UnassignedAccountHost[] }> {
  const { data } = await apiClient.get<{ items: UnassignedAccountHost[] }>(
    '/admin/upstreams/unassigned-accounts'
  )
  return data
}

/** 批量把账号挂到上游 */
export async function assignAccounts(id: number, accountIds: number[]): Promise<void> {
  await apiClient.put(`/admin/upstreams/${id}/accounts`, { account_ids: accountIds })
}

/** 新建/更新上游侧分组 */
export async function upsertGroup(id: number, input: UpstreamGroupInput): Promise<UpstreamGroup> {
  const { data } = await apiClient.post<UpstreamGroup>(`/admin/upstreams/${id}/groups`, input)
  return data
}

/** 删除上游侧分组 */
export async function deleteGroup(id: number, groupId: number): Promise<void> {
  await apiClient.delete(`/admin/upstreams/${id}/groups/${groupId}`)
}

/** 新建/更新台账条目 */
export async function upsertModelPrice(
  id: number,
  input: UpstreamModelPriceInput
): Promise<UpstreamModelPrice> {
  const { data } = await apiClient.post<UpstreamModelPrice>(
    `/admin/upstreams/${id}/model-prices`,
    input
  )
  return data
}

/** 删除台账条目 */
export async function deleteModelPrice(id: number, priceId: number): Promise<void> {
  await apiClient.delete(`/admin/upstreams/${id}/model-prices/${priceId}`)
}

/** 批量导入台账（幂等 upsert） */
export async function importModelPrices(
  id: number,
  items: UpstreamModelPriceInput[],
  source: 'import' | 'crawler' = 'import'
): Promise<UpstreamImportResult> {
  const { data } = await apiClient.post<UpstreamImportResult>(
    `/admin/upstreams/${id}/model-prices/import`,
    { source, items }
  )
  return data
}

/** 上游整包导入（上游 + 分组 + 台账） */
export async function importBundle(
  bundle: UpstreamBundleImport,
  source: 'import' | 'crawler' = 'import'
): Promise<{ upstream: Upstream; import: UpstreamImportResult }> {
  const { data } = await apiClient.post<{ upstream: Upstream; import: UpstreamImportResult }>(
    `/admin/upstreams/import-bundle`,
    bundle,
    { params: { source } }
  )
  return data
}

/** 跨上游比价矩阵 */
export async function compare(params?: {
  platform?: string
  search?: string
}): Promise<UpstreamCompareResult> {
  const { data } = await apiClient.get<UpstreamCompareResult>('/admin/upstreams/compare', { params })
  return data
}

const upstreamsAPI = {
  list,
  create,
  getById,
  update,
  remove,
  unassignedAccounts,
  assignAccounts,
  upsertGroup,
  deleteGroup,
  upsertModelPrice,
  deleteModelPrice,
  importModelPrices,
  importBundle,
  compare
}

export default upstreamsAPI
