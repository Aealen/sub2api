<template>
  <DataTable
    :columns="columns"
    :data="rows"
    :loading="loading && rows.length === 0"
    :row-key="rowKeyOf"
  >
    <template #empty>
      <div class="py-10 text-center text-sm text-gray-400">
        {{ t('admin.upstreams.overview.empty') }}
      </div>
    </template>

    <!-- 首列：本站分组（按本站分组视角）/ 上游（按上游视角） -->
    <template #cell-group="{ row }">
      <div
        class="min-w-[150px]"
        :class="view === 'group'
          ? ['border-l-2 border-primary-500 pl-2', row.gapTop ? 'mt-1' : '']
          : ''"
      >
        <div
          v-if="row.kind !== 'upstream' && (view !== 'group' || row.prevGroupId !== row.group_id)"
          class="font-medium text-gray-900 dark:text-white"
        >
          {{ row.group_name }}
        </div>
      </div>
    </template>

    <template #cell-platform="{ row }">
      <span v-if="row.platform" class="text-xs text-gray-600 dark:text-gray-300">
        {{ row.platform }}
      </span>
    </template>

    <template #cell-groupRate="{ row }">
      <span v-if="row.kind !== 'upstream'" class="font-mono text-sm">
        {{ formatRate(row.group_rate_multiplier) }}
      </span>
    </template>

    <template #cell-upstream="{ row }">
      <div
        class="min-w-[140px]"
        :class="view === 'upstream'
          ? ['border-l-2 border-primary-500 pl-2', row.gapTop ? 'mt-1' : '']
          : ''"
      >
        <div class="font-medium text-gray-900 dark:text-white">{{ row.upstream_name }}</div>
        <div v-if="row.kind === 'upstream' && row.group_count != null" class="text-xs text-gray-400">
          {{ t('admin.upstreams.overview.balance.groups') }} {{ row.group_count }}
        </div>
      </div>
    </template>

    <template #cell-status="{ row }">
      <span
        v-if="row.kind === 'upstream'"
        class="inline-flex rounded-full px-2 py-0.5 text-xs"
        :class="row.status === 'active'
          ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
          : 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300'"
      >
        {{ t(`admin.upstreams.status.${row.status}`) }}
      </span>
    </template>

    <template #cell-balance="{ row }">
      <span
        v-if="row.kind === 'upstream'"
        class="font-mono text-sm"
        :class="row.site_balance <= 0
          ? 'text-gray-400 dark:text-dark-400'
          : 'text-gray-900 dark:text-gray-100'"
      >
        {{ formatMoney(row.site_balance) }}
      </span>
    </template>

    <template #cell-upstreamGroup="{ row }">
      <span v-if="row.linkId != null" class="text-xs text-gray-600 dark:text-gray-300">
        {{ row.upstream_group_name }}
      </span>
    </template>

    <template #cell-purchaseRate="{ row }">
      <!-- 进价属于「关联」而非「行视角」：两种视角的子行都可就地改上游分组倍率 -->
      <span v-if="row.linkId != null" class="inline-flex items-center gap-1.5">
        <span
          class="font-mono text-sm"
          :class="row.upstream_rate_multiplier == null ? 'text-gray-400' : 'text-gray-900 dark:text-gray-100'"
        >
          {{ formatRate(row.upstream_rate_multiplier) }}
        </span>
        <span
          v-if="row.isPrimary"
          class="rounded bg-primary-100 px-1.5 py-0.5 text-[10px] font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
        >
          {{ t('admin.upstreams.overview.primary') }}
        </span>
        <button
          v-if="row.upstream_group_id != null"
          type="button"
          class="rounded p-0.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-600 dark:hover:text-primary-400"
          :title="t('admin.upstreams.overview.rateDialog.title')"
          :aria-label="t('admin.upstreams.overview.rateDialog.title')"
          :data-testid="`upstream-overview-edit-rate-${row.linkId}`"
          @click.stop="editRate(row)"
        >
          <Icon name="edit" size="sm" />
        </button>
      </span>
    </template>

    <template #cell-derivedMargin="{ row }">
      <template v-if="row.linkId != null">
        <span v-if="row.derived_margin_pct == null" class="text-xs text-gray-400">
          {{ t('admin.upstreams.overview.noData') }}
        </span>
        <span v-else class="font-mono text-sm" :class="marginClass(row.derived_margin_pct)">
          {{ formatPercent(row.derived_margin_pct) }}
        </span>
      </template>
    </template>

    <template #cell-actualProfit="{ row }">
      <template v-if="row.linkId != null">
        <span
          v-if="row.flow"
          class="font-mono text-sm"
          :class="row.flow.profit < 0 ? 'text-red-600 dark:text-red-400' : 'text-gray-900 dark:text-gray-100'"
        >
          {{ formatMoney(row.flow.profit) }}
        </span>
        <span v-else class="text-xs text-gray-400">
          {{ t('admin.upstreams.overview.noFlow') }}
        </span>
      </template>
    </template>

    <template #cell-actualMargin="{ row }">
      <template v-if="row.linkId != null">
        <span
          v-if="row.flow && row.flow.margin_pct != null"
          class="font-mono text-sm"
          :class="marginClass(row.flow.margin_pct)"
        >
          {{ formatPercent(row.flow.margin_pct) }}
        </span>
        <span v-else-if="row.flow" class="text-xs text-gray-400">
          {{ t('admin.upstreams.overview.noData') }}
        </span>
        <span v-else class="text-xs text-gray-400">
          {{ t('admin.upstreams.overview.noFlow') }}
        </span>
      </template>
    </template>

    <template #cell-accounts="{ row }">
      <span v-if="row.kind === 'upstream'" class="font-mono text-sm">
        {{ row.account_active }}/{{ row.account_total }}
      </span>
      <span v-else class="font-mono text-sm">{{ row.bound_account_count }}</span>
    </template>

    <template #cell-actions="{ row }">
      <button
        v-if="row.linkId != null"
        type="button"
        class="text-xs text-red-500 hover:underline"
        :data-testid="`upstream-overview-unlink-${row.linkId}`"
        @click="askUnlink(row)"
      >
        {{ t('admin.upstreams.overview.link.unlink') }}
      </button>
    </template>
  </DataTable>

  <ConfirmDialog
    :show="unlinkVisible"
    :title="t('admin.upstreams.overview.link.confirmTitle')"
    :message="unlinkMessage"
    :confirm-text="t('admin.upstreams.overview.link.unlink')"
    :cancel-text="t('common.cancel')"
    :danger="true"
    @confirm="confirmUnlink"
    @cancel="closeUnlink"
  />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import DataTable from '@/components/common/DataTable.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { Column } from '@/components/common/types'
import type {
  UpstreamFlowRow,
  UpstreamOverviewLink,
  UpstreamOverviewResult
} from '@/api/admin/upstreams'

type OverviewView = 'group' | 'upstream'
type RowKind = 'link' | 'upstream' | 'upstreamGroup'

const props = withDefaults(defineProps<{
  result: UpstreamOverviewResult | null
  view: OverviewView
  windowDays: number
  keyword: string
  loading?: boolean
}>(), {
  loading: false
})

/** 就地编辑进价时交给弹窗的上下文：上游 + 上游分组 + 当前倍率 */
interface EditRatePayload {
  upstreamId: number
  upstreamGroupId: number
  upstreamName: string
  upstreamGroupName: string
  rateMultiplier: number | null
}

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'edit-rate', payload: EditRatePayload): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const unlinkVisible = ref(false)
const unlinking = ref<TableRow | null>(null)

/**
 * 展平后的表格行：两种视角共用同一套列 key，
 * 与视角无关的字段留空，由各 cell 插槽按 kind 决定是否渲染。
 */
interface TableRow {
  rowKey: string
  kind: RowKind
  /** 供货关联 id：只有关联行才有，解除关联时使用 */
  linkId: number | null
  group_id: number | null
  group_name: string | null
  platform: string | null
  group_rate_multiplier: number | null
  upstream_id: number | null
  upstream_name: string | null
  status: string | null
  site_balance: number | null
  account_active: number | null
  account_total: number | null
  group_count: number | null
  /** 上游侧分组 id：就地编辑进价倍率时定位到具体分组 */
  upstream_group_id: number | null
  upstream_group_name: string | null
  upstream_rate_multiplier: number | null
  derived_margin_pct: number | null
  bound_account_count: number | null
  /** 上一行所属的本站分组 id：仅在「按本站分组」视角用于判断组内首行 */
  prevGroupId: number | null
  /** 新分组的首行：隔开相邻分组的左侧主色边框，避免连成一条线 */
  gapTop: boolean
  /** 组内进价最低的一行（主力供货） */
  isPrimary: boolean
  flow: UpstreamFlowRow | null
}

const makeBaseRow = (): TableRow => ({
  rowKey: '',
  kind: 'link',
  linkId: null,
  group_id: null,
  group_name: null,
  platform: null,
  group_rate_multiplier: null,
  upstream_id: null,
  upstream_name: null,
  status: null,
  site_balance: null,
  account_active: null,
  account_total: null,
  group_count: null,
  upstream_group_id: null,
  upstream_group_name: null,
  upstream_rate_multiplier: null,
  derived_margin_pct: null,
  bound_account_count: null,
  prevGroupId: null,
  gapTop: false,
  isPrimary: false,
  flow: null
})

const columns = computed<Column[]>(() => {
  const base: Column[] = [
    { key: 'group', label: t('admin.upstreams.overview.columns.group'), sortable: false },
    { key: 'platform', label: t('admin.upstreams.overview.columns.platform'), sortable: false },
    { key: 'groupRate', label: t('admin.upstreams.overview.columns.groupRate'), sortable: false },
    { key: 'upstream', label: t('admin.upstreams.overview.columns.upstream'), sortable: false },
    { key: 'upstreamGroup', label: t('admin.upstreams.overview.columns.upstreamGroup'), sortable: false },
    { key: 'purchaseRate', label: t('admin.upstreams.overview.columns.purchaseRate'), sortable: false },
    { key: 'derivedMargin', label: t('admin.upstreams.overview.columns.derivedMargin'), sortable: false },
    { key: 'actualProfit', label: t('admin.upstreams.overview.columns.actualProfit'), sortable: false },
    { key: 'actualMargin', label: t('admin.upstreams.overview.columns.actualMargin'), sortable: false },
    { key: 'accounts', label: t('admin.upstreams.overview.columns.accounts'), sortable: false }
  ]
  const actions: Column = {
    key: 'actions',
    label: t('admin.upstreams.overview.columns.actions'),
    sortable: false
  }

  if (props.view === 'group') return [...base, actions]

  return [
    base[3], // 上游
    { key: 'status', label: t('admin.upstreams.overview.columns.status'), sortable: false },
    { key: 'balance', label: t('admin.upstreams.overview.columns.balance'), sortable: false },
    { key: 'accounts', label: t('admin.upstreams.overview.columns.accounts'), sortable: false },
    base[0], // 本站分组
    base[1], // 平台
    base[2], // 售价 D
    base[4], // 上游分组
    base[5], // 进价 U'
    base[6], // 推算毛利率
    base[7], // 实际毛利
    base[8], // 实际毛利率
    actions
  ]
})

const flowKey = (groupId: number, upstreamId: number): string => `${groupId}:${upstreamId}`

/** 当前窗口的「本站分组 × 上游」真实流水；窗口不匹配的流水直接丢弃 */
const flowMap = computed(() => {
  const map = new Map<string, UpstreamFlowRow>()
  for (const row of props.result?.flow ?? []) {
    if (row.window_days !== props.windowDays) continue
    map.set(flowKey(row.group_id, row.upstream_id), row)
  }
  return map
})

const flowFor = (groupId: number, upstreamId: number): UpstreamFlowRow | null =>
  flowMap.value.get(flowKey(groupId, upstreamId)) ?? null

/** 组内按进价升序；进价缺失排最后，同价按上游名稳定排序 */
const comparePurchaseRate = (a: UpstreamOverviewLink, b: UpstreamOverviewLink): number => {
  const left = a.upstream_rate_multiplier
  const right = b.upstream_rate_multiplier
  if (left == null && right == null) return a.upstream_name.localeCompare(b.upstream_name)
  if (left == null) return 1
  if (right == null) return -1
  if (left !== right) return left - right
  return a.upstream_name.localeCompare(b.upstream_name)
}

const matchesKeyword = (link: UpstreamOverviewLink, keyword: string): boolean =>
  link.group_name.toLowerCase().includes(keyword) ||
  link.upstream_name.toLowerCase().includes(keyword)

/** 按本站分组视角：同一 group_id 的行连续，组内首行才显示分组名 */
const groupRows = computed<TableRow[]>(() => {
  const keyword = props.keyword.trim().toLowerCase()
  const buckets = new Map<number, UpstreamOverviewLink[]>()
  for (const link of props.result?.links ?? []) {
    if (keyword && !matchesKeyword(link, keyword)) continue
    const bucket = buckets.get(link.group_id)
    if (bucket) bucket.push(link)
    else buckets.set(link.group_id, [link])
  }

  const rows: TableRow[] = []
  /** 跨分组保留上一个 group_id，用于在分组切换处加分隔间距（必须声明在循环外，否则恒为新分组首行） */
  let prevGroupId: number | null = null
  for (const groupLinks of buckets.values()) {
    const sorted = [...groupLinks].sort(comparePurchaseRate)
    const primaryLink = sorted.find((link) => link.upstream_rate_multiplier != null) ?? null
    for (const link of sorted) {
      rows.push({
        ...makeBaseRow(),
        rowKey: `link:${link.id}`,
        kind: 'link',
        linkId: link.id,
        group_id: link.group_id,
        group_name: link.group_name,
        platform: link.platform,
        group_rate_multiplier: link.group_rate_multiplier,
        upstream_id: link.upstream_id,
        upstream_name: link.upstream_name,
        upstream_group_id: link.upstream_group_id,
        upstream_group_name: link.upstream_group_name,
        upstream_rate_multiplier: link.upstream_rate_multiplier,
        derived_margin_pct: link.derived_margin_pct,
        bound_account_count: link.bound_account_count,
        prevGroupId,
        gapTop: prevGroupId !== null && prevGroupId !== link.group_id,
        isPrimary: primaryLink != null && primaryLink.id === link.id,
        flow: flowFor(link.group_id, link.upstream_id)
      })
      prevGroupId = link.group_id
    }
  }
  return rows
})

/** 按上游视角：主行是上游（余额 / 账号数），子行是该上游供货的本站分组 */
const upstreamRows = computed<TableRow[]>(() => {
  const keyword = props.keyword.trim().toLowerCase()
  const linksByUpstream = new Map<number, UpstreamOverviewLink[]>()
  for (const link of props.result?.links ?? []) {
    const bucket = linksByUpstream.get(link.upstream_id)
    if (bucket) bucket.push(link)
    else linksByUpstream.set(link.upstream_id, [link])
  }

  const rows: TableRow[] = []
  for (const upstream of props.result?.upstreams ?? []) {
    const allLinks = linksByUpstream.get(upstream.id) ?? []
    const upstreamMatches = !keyword || upstream.name.toLowerCase().includes(keyword)
    const matchingLinks = keyword ? allLinks.filter((link) => matchesKeyword(link, keyword)) : allLinks
    if (keyword && !upstreamMatches && matchingLinks.length === 0) continue

    rows.push({
      ...makeBaseRow(),
      rowKey: `upstream:${upstream.id}`,
      kind: 'upstream',
      upstream_id: upstream.id,
      upstream_name: upstream.name,
      status: upstream.status,
      site_balance: upstream.site_balance,
      account_active: upstream.account_active,
      account_total: upstream.account_total,
      group_count: upstream.group_count,
      gapTop: rows.length > 0
    })

    const children = [...(upstreamMatches ? allLinks : matchingLinks)].sort(comparePurchaseRate)
    for (const link of children) {
      rows.push({
        ...makeBaseRow(),
        rowKey: `upstream-group:${upstream.id}:${link.id}`,
        kind: 'upstreamGroup',
        linkId: link.id,
        group_id: link.group_id,
        group_name: link.group_name,
        platform: link.platform,
        group_rate_multiplier: link.group_rate_multiplier,
        upstream_id: link.upstream_id,
        upstream_name: link.upstream_name,
        upstream_group_id: link.upstream_group_id,
        upstream_group_name: link.upstream_group_name,
        upstream_rate_multiplier: link.upstream_rate_multiplier,
        derived_margin_pct: link.derived_margin_pct,
        bound_account_count: link.bound_account_count,
        flow: flowFor(link.group_id, link.upstream_id)
      })
    }
  }
  return rows
})

const rows = computed<TableRow[]>(() =>
  props.view === 'group' ? groupRows.value : upstreamRows.value
)

const rowKeyOf = (row: any): string => String(row?.rowKey ?? '')

const unlinkMessage = computed(() => {
  const target = unlinking.value
  if (!target) return ''
  return t('admin.upstreams.overview.link.confirmMessage', {
    group: target.group_name ?? '',
    upstream: target.upstream_name ?? '',
    upstreamGroup: target.upstream_group_name ?? ''
  })
})

const askUnlink = (row: TableRow) => {
  if (row.linkId == null) return
  unlinking.value = row
  unlinkVisible.value = true
}

const closeUnlink = () => {
  unlinkVisible.value = false
  unlinking.value = null
}

/** 打开「编辑上游进价倍率」弹窗：缺少上游或上游分组定位信息时直接忽略 */
const editRate = (row: TableRow) => {
  if (row.upstream_id == null || row.upstream_group_id == null) return
  emit('edit-rate', {
    upstreamId: row.upstream_id,
    upstreamGroupId: row.upstream_group_id,
    upstreamName: row.upstream_name ?? '',
    upstreamGroupName: row.upstream_group_name ?? '',
    rateMultiplier: row.upstream_rate_multiplier
  })
}

const confirmUnlink = async () => {
  const linkId = unlinking.value?.linkId ?? null
  closeUnlink()
  if (linkId == null) return
  try {
    await adminAPI.upstreams.deleteGroupLink(linkId)
    appStore.showSuccess(t('admin.upstreams.overview.link.unlinked'))
    emit('refresh')
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.overview.link.unlinkFailed'))
  }
}

const formatRate = (value: number | null | undefined): string => {
  if (value == null || !Number.isFinite(value)) return '—'
  return `×${Number(value.toFixed(4))}`
}

const formatPercent = (value: number | null | undefined): string => {
  if (value == null || !Number.isFinite(value)) return '—'
  return `${value.toFixed(1)}%`
}

const formatMoney = (value: number | null | undefined): string => {
  if (value == null || !Number.isFinite(value)) return '—'
  return `$${value.toFixed(2)}`
}

/** 毛利率配色：>=20 绿、10~20 黄、<10 橙、<=0 红 */
const marginClass = (value: number | null | undefined): string => {
  if (value == null || !Number.isFinite(value)) return 'text-gray-400'
  if (value <= 0) return 'text-red-600 dark:text-red-400'
  if (value < 10) return 'text-orange-600 dark:text-orange-400'
  if (value < 20) return 'text-yellow-600 dark:text-yellow-400'
  return 'text-green-600 dark:text-green-400'
}
</script>
