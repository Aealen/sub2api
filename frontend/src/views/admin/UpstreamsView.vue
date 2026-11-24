<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-start">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-64">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="search"
                type="text"
                :placeholder="t('admin.upstreams.searchPlaceholder')"
                class="input pl-10"
                data-testid="upstream-search"
              />
            </div>
            <Select
              v-model="statusFilter"
              :options="statusOptions"
              :placeholder="t('admin.upstreams.allStatus')"
              class="w-40"
              data-testid="upstream-status-filter"
            />
            <button
              v-if="unassignedCount > 0"
              class="rounded-lg border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-700 transition-colors hover:bg-amber-100 dark:border-amber-500/40 dark:bg-amber-900/20 dark:text-amber-300"
              data-testid="upstream-unassigned-banner"
              @click="unassignedVisible = true"
            >
              {{ t('admin.upstreams.unassignedBanner', { count: unassignedCount }) }}
            </button>
          </div>

          <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-3 lg:w-auto">
            <button class="btn btn-secondary" :disabled="loading" :title="t('common.refresh')" @click="load">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <router-link to="/admin/upstreams/compare" class="btn btn-secondary">
              <Icon name="chart" size="md" class="mr-2" />
              {{ t('admin.upstreams.compareEntry') }}
            </router-link>
            <button class="btn btn-primary" data-testid="upstream-create" @click="openCreate">
              <Icon name="plus" size="md" class="mr-2" />
              {{ t('admin.upstreams.create') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="filteredItems"
          :loading="loading && items.length === 0"
          row-key="id"
        >
          <template #empty>
            <div class="py-10 text-center text-sm text-gray-400">{{ t('admin.upstreams.empty') }}</div>
          </template>
          <template #cell-name="{ row }">
            <div class="min-w-[140px]">
              <div class="font-medium text-gray-900 dark:text-white">{{ row.name }}</div>
              <div v-if="row.base_url" class="text-xs text-gray-400">{{ row.base_url }}</div>
            </div>
          </template>

          <template #cell-status="{ row }">
            <span
              class="inline-flex rounded-full px-2 py-0.5 text-xs"
              :class="row.status === 'active'
                ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
                : 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300'"
            >
              {{ t(`admin.upstreams.status.${row.status}`) }}
            </span>
          </template>

          <template #cell-accounts="{ row }">
            <span class="font-mono text-sm">{{ row.account_active }}/{{ row.account_total }}</span>
          </template>

          <template #cell-balance="{ row }">
            <div class="font-mono text-sm" :class="row.site_balance <= 0 ? 'text-gray-400' : 'text-gray-900 dark:text-gray-100'">
              ${{ row.site_balance.toFixed(2) }}
            </div>
            <div class="text-xs text-gray-400">
              {{ row.balance_updated_at ? formatTime(row.balance_updated_at) : '—' }}
            </div>
          </template>

          <template #cell-rateRange="{ row }">
            <span class="font-mono text-sm">{{ formatRateRange(row) }}</span>
          </template>

          <template #cell-models="{ row }">
            <span class="font-mono text-sm">{{ row.model_price_count }}</span>
          </template>

          <template #cell-groups="{ row }">
            <span class="text-xs text-gray-600 dark:text-gray-300">
              {{ row.group_names.length > 0 ? row.group_names.join(', ') : '—' }}
            </span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center justify-end gap-3 whitespace-nowrap">
              <button class="text-xs text-primary-600 hover:underline" @click="openDetail(row.id)">
                {{ t('admin.upstreams.actions.detail') }}
              </button>
              <button class="text-xs text-gray-600 hover:underline dark:text-gray-300" @click="openEdit(row)">
                {{ t('admin.upstreams.actions.edit') }}
              </button>
              <button class="text-xs text-red-500 hover:underline" @click="askDelete(row)">
                {{ t('admin.upstreams.actions.delete') }}
              </button>
            </div>
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <UpstreamFormDialog v-model:show="formVisible" :upstream="editing" @saved="onSaved" />
    <UpstreamDetailDialog
      v-model:show="detailVisible"
      :upstream-id="detailId"
      @changed="load"
    />
    <UnassignedAccountsDialog
      v-model:show="unassignedVisible"
      :upstreams="items"
      @assigned="onAssigned"
    />
    <ConfirmDialog
      :show="deleteVisible"
      :title="t('admin.upstreams.actions.delete')"
      :message="deleteMessage"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="deleteVisible = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Select from '@/components/common/Select.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import UpstreamFormDialog from '@/components/admin/upstream/UpstreamFormDialog.vue'
import UpstreamDetailDialog from '@/components/admin/upstream/UpstreamDetailDialog.vue'
import UnassignedAccountsDialog from '@/components/admin/upstream/UnassignedAccountsDialog.vue'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { Column } from '@/components/common/types'
import type { Upstream, UpstreamOverview } from '@/api/admin/upstreams'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const items = ref<UpstreamOverview[]>([])
const unassignedCount = ref(0)
const search = ref('')
const statusFilter = ref<string | null>(null)

const formVisible = ref(false)
const editing = ref<Upstream | null>(null)
const detailVisible = ref(false)
const detailId = ref<number | null>(null)
const unassignedVisible = ref(false)
const deleteVisible = ref(false)
const deleting = ref<UpstreamOverview | null>(null)

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.upstreams.columns.name'), sortable: false },
  { key: 'status', label: t('admin.upstreams.columns.status'), sortable: false },
  { key: 'accounts', label: t('admin.upstreams.columns.accounts'), sortable: false },
  { key: 'balance', label: t('admin.upstreams.columns.balance'), sortable: false },
  { key: 'rateRange', label: t('admin.upstreams.columns.rateRange'), sortable: false },
  { key: 'models', label: t('admin.upstreams.columns.models'), sortable: false },
  { key: 'groups', label: t('admin.upstreams.columns.groups'), sortable: false },
  { key: 'actions', label: t('admin.upstreams.columns.actions'), sortable: false }
])

const statusOptions = computed(() => [
  { value: null, label: t('admin.upstreams.allStatus') },
  { value: 'active', label: t('admin.upstreams.status.active') },
  { value: 'inactive', label: t('admin.upstreams.status.inactive') }
])

const filteredItems = computed(() => {
  const keyword = search.value.trim().toLowerCase()
  return items.value.filter((item) => {
    if (statusFilter.value && item.status !== statusFilter.value) return false
    if (!keyword) return true
    return (
      item.name.toLowerCase().includes(keyword) ||
      item.base_url.toLowerCase().includes(keyword)
    )
  })
})

const deleteMessage = computed(() =>
  deleting.value ? t('admin.upstreams.messages.confirmDelete', { name: deleting.value.name }) : ''
)

const formatTime = (raw: string): string => {
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return '—'
  const diffMinutes = Math.floor((Date.now() - date.getTime()) / 60000)
  if (diffMinutes < 1) return t('common.justNow')
  if (diffMinutes < 60) return `${diffMinutes}m`
  if (diffMinutes < 1440) return `${Math.floor(diffMinutes / 60)}h`
  return date.toLocaleDateString()
}

const formatRateRange = (row: UpstreamOverview): string => {
  if (row.rate_multiplier_min == null && row.rate_multiplier_max == null) return '—'
  const min = row.rate_multiplier_min ?? row.rate_multiplier_max
  const max = row.rate_multiplier_max ?? row.rate_multiplier_min
  if (min == null || max == null) return '—'
  return min === max ? `×${min}` : `×${min}~×${max}`
}

const load = async () => {
  loading.value = true
  try {
    const [result, unassigned] = await Promise.all([
      adminAPI.upstreams.list(),
      adminAPI.upstreams.unassignedAccounts().catch(() => ({ items: [] }))
    ])
    items.value = result.items
    unassignedCount.value = unassigned.items.reduce((sum, host) => sum + host.account_count, 0)
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.messages.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(load)

const openCreate = () => {
  editing.value = null
  formVisible.value = true
}

const openEdit = (row: UpstreamOverview) => {
  editing.value = row
  formVisible.value = true
}

const onSaved = async () => {
  await load()
}

const openDetail = (id: number) => {
  detailId.value = id
  detailVisible.value = true
}

const onAssigned = async () => {
  await load()
}

const askDelete = (row: UpstreamOverview) => {
  deleting.value = row
  deleteVisible.value = true
}

const confirmDelete = async () => {
  if (!deleting.value) return
  const target = deleting.value
  deleteVisible.value = false
  try {
    await adminAPI.upstreams.remove(target.id)
    appStore.showSuccess(t('admin.upstreams.messages.deleted'))
    await load()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.messages.deleteFailed'))
  }
}
</script>
