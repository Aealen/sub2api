<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-start">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-56">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="search"
                type="text"
                :placeholder="t('admin.upstreams.compare.searchPlaceholder')"
                class="input pl-10"
                data-testid="upstream-compare-search"
                @input="onSearchInput"
              />
            </div>
            <MultiSelectDropdown
              v-model="modelSelection"
              :options="modelOptions"
              :placeholder="t('admin.upstreams.compare.allModels')"
              test-id="upstream-compare-model-filter"
              class="w-44"
            />
            <MultiSelectDropdown
              v-model="upstreamSelection"
              :options="upstreamFilterOptions"
              :placeholder="t('admin.upstreams.compare.allUpstreams')"
              :searchable="false"
              test-id="upstream-compare-upstream-filter"
              class="w-40"
            />
            <MultiSelectDropdown
              v-model="groupSelection"
              :options="groupFilterOptions"
              :placeholder="t('admin.upstreams.compare.allGroups')"
              test-id="upstream-compare-group-filter"
              class="w-48"
            />
            <Select
              v-model="platformFilter"
              :options="platformOptions"
              :placeholder="t('admin.upstreams.compare.allPlatforms')"
              class="w-36"
              data-testid="upstream-compare-platform"
              @change="load"
            />
          </div>

          <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-3 lg:w-auto">
            <button class="btn btn-secondary" :disabled="loading" :title="t('common.refresh')" @click="load">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <router-link to="/admin/upstreams" class="btn btn-secondary">
              <Icon name="arrowLeft" size="md" class="mr-2" />
              {{ t('nav.upstreamList') }}
            </router-link>
          </div>
        </div>
        <p class="mt-3 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.upstreams.compare.description') }}</p>
      </template>

      <template #table>
        <div v-if="loading" class="flex items-center justify-center py-10">
          <LoadingSpinner />
        </div>

        <div v-else-if="!result || result.rows.length === 0" class="py-12 text-center text-sm text-gray-400">
          {{ t('admin.upstreams.compare.empty') }}
        </div>

        <div v-else-if="filteredColumns.length === 0" class="py-12 text-center text-sm text-gray-400">
          {{ t('admin.upstreams.compare.noColumns') }}
        </div>

        <div v-else class="overflow-x-auto">
          <table class="min-w-full border-separate border-spacing-0 text-sm" data-testid="upstream-compare-table">
            <thead>
              <!-- 二级表头：第一级 = 供应商（colspan），第二级 = 分组 ×倍率 -->
              <tr class="text-left text-xs uppercase text-gray-500 dark:text-gray-400">
                <th rowspan="2" class="sticky left-0 z-10 bg-white px-3 py-2 dark:bg-dark-800">{{ t('admin.upstreams.compare.model') }}</th>
                <th rowspan="2" class="px-3 py-2">{{ t('admin.upstreams.compare.official') }}</th>
                <th
                  v-for="colGroup in columnGroups"
                  :key="colGroup.upstreamID"
                  :colspan="colGroup.cols.length"
                  class="border-b border-gray-200 whitespace-nowrap px-3 py-2 text-center font-semibold text-gray-700 dark:border-dark-600 dark:text-gray-200"
                >
                  {{ colGroup.upstreamName }}
                </th>
              </tr>
              <tr class="text-left text-xs text-gray-500 dark:text-gray-400">
                <th
                  v-for="col in filteredColumns"
                  :key="columnKey(col)"
                  class="border-b border-gray-100 whitespace-nowrap px-3 py-2 font-mono dark:border-dark-600"
                  :class="col.kind === 'derived' && 'italic'"
                >
                  {{ col.group_name }}
                  <span v-if="col.rate_multiplier != null" class="text-gray-400">×{{ col.rate_multiplier }}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="row in filteredRows"
                :key="row.model_name"
                class="border-t border-gray-100 dark:border-dark-600"
              >
                <td class="sticky left-0 z-10 bg-white px-3 py-2 dark:bg-dark-800">
                  <div class="font-mono text-xs font-medium text-gray-900 dark:text-white">{{ row.model_name }}</div>
                  <div v-if="row.platform" class="text-xs text-gray-400">{{ row.platform }}</div>
                </td>
                <td class="whitespace-nowrap px-3 py-2 font-mono text-xs text-gray-500 dark:text-gray-400">
                  {{ formatPerMillion(row.official_input_price_per_token) }}
                </td>
                <td
                  v-for="col in filteredColumns"
                  :key="columnKey(col)"
                  class="whitespace-nowrap px-3 py-2"
                >
                  <template v-if="cellFor(row, col)">
                    <div
                      class="inline-flex flex-col rounded-md px-2 py-1"
                      :class="isCheapest(row, col)
                        ? 'bg-green-50 ring-1 ring-green-300 dark:bg-green-900/20 dark:ring-green-700'
                        : ''"
                      :data-testid="`upstream-compare-cell-${columnKey(col)}`"
                    >
                      <span class="font-mono text-xs font-medium text-gray-900 dark:text-gray-100">
                        {{ formatPerMillion(cellFor(row, col)?.input_price_per_token) }}
                      </span>
                      <span class="text-[10px] text-gray-400">
                        <template v-if="cellFor(row, col)?.rate_multiplier != null">
                          ×{{ cellFor(row, col)?.rate_multiplier }}
                        </template>
                        <span
                          class="ml-1 rounded px-1"
                          :class="cellFor(row, col)?.source === 'ledger'
                            ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                            : 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300'"
                        >
                          {{ cellFor(row, col)?.source === 'ledger'
                            ? t('admin.upstreams.compare.sourceLedger')
                            : t('admin.upstreams.compare.sourceDerived') }}
                        </span>
                      </span>
                      <span
                        v-if="cellFor(row, col)?.account_name"
                        class="text-[10px] text-gray-400"
                      >
                        {{ cellFor(row, col)?.account_name }}
                      </span>
                    </div>
                  </template>
                  <span v-else class="text-xs text-gray-300 dark:text-dark-500">{{ t('admin.upstreams.compare.noData') }}</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Select from '@/components/common/Select.vue'
import MultiSelectDropdown from '@/components/common/MultiSelectDropdown.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { UpstreamCompareColumn, UpstreamCompareResult, UpstreamCompareRow } from '@/api/admin/upstreams'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const result = ref<UpstreamCompareResult | null>(null)
const platformFilter = ref<string | null>(null)
const search = ref('')
const modelSelection = ref<(string | number)[]>([])
const upstreamSelection = ref<(string | number)[]>([])
const groupSelection = ref<(string | number)[]>([])
let searchTimer: ReturnType<typeof setTimeout> | null = null

/** 列的唯一 key：上游 id + 分组 id（综合列分组为 null → x） */
const columnKey = (col: UpstreamCompareColumn): string =>
  `${col.upstream_id}:${col.group_id ?? 'x'}`

/** 供应商多选选项（来自全部列，不随筛选变化） */
const upstreamFilterOptions = computed(() => {
  const seen = new Map<number, string>()
  for (const col of result.value?.columns ?? []) {
    if (!seen.has(col.upstream_id)) seen.set(col.upstream_id, col.upstream_name)
  }
  return Array.from(seen, ([value, label]) => ({ value, label }))
})

/** 分组多选选项：label = 供应商 / 分组（含综合列） */
const groupFilterOptions = computed(() =>
  (result.value?.columns ?? []).map((col) => ({
    value: columnKey(col),
    label: `${col.upstream_name} / ${col.group_name}`
  }))
)

/** 模型多选选项（来自全部行，不随筛选变化） */
const modelOptions = computed(() =>
  (result.value?.rows ?? []).map((row) => ({ value: row.model_name, label: row.model_name }))
)

/** 列筛选：供应商多选（空=全部）AND 分组多选（空=全部，按列 key） */
const filteredColumns = computed<UpstreamCompareColumn[]>(() => {
  const cols = result.value?.columns ?? []
  return cols.filter((col) => {
    if (upstreamSelection.value.length > 0 && !upstreamSelection.value.includes(col.upstream_id)) {
      return false
    }
    if (groupSelection.value.length > 0 && !groupSelection.value.includes(columnKey(col))) {
      return false
    }
    return true
  })
})

/** 二级表头第一级：按供应商聚合筛选后的列 */
const columnGroups = computed(() => {
  const groups: { upstreamID: number; upstreamName: string; cols: UpstreamCompareColumn[] }[] = []
  for (const col of filteredColumns.value) {
    const last = groups[groups.length - 1]
    if (last && last.upstreamID === col.upstream_id) {
      last.cols.push(col)
    } else {
      groups.push({ upstreamID: col.upstream_id, upstreamName: col.upstream_name, cols: [col] })
    }
  }
  return groups
})

/** 行筛选：文本搜索 + 模型多选（空=全部）；平台过滤在后端请求时带参 */
const filteredRows = computed<UpstreamCompareRow[]>(() => {
  const rows = result.value?.rows ?? []
  const keyword = search.value.trim().toLowerCase()
  const selectedModels = modelSelection.value as string[]
  return rows.filter((row) => {
    if (selectedModels.length > 0 && !selectedModels.includes(row.model_name)) return false
    if (keyword && !row.model_name.toLowerCase().includes(keyword)) return false
    return true
  })
})

const cellFor = (row: UpstreamCompareRow, col: UpstreamCompareColumn) =>
  row.cells?.find(
    (cell) =>
      cell.upstream_id === col.upstream_id &&
      (cell.group_id ?? null) === (col.group_id ?? null)
  ) ?? null

const isCheapest = (row: UpstreamCompareRow, col: UpstreamCompareColumn): boolean =>
  row.cheapest_upstream_id === col.upstream_id &&
  (row.cheapest_group_id ?? null) === (col.group_id ?? null)

const platformOptions = computed(() => {
  const platforms = new Set<string>()
  for (const row of result.value?.rows ?? []) {
    if (row.platform) platforms.add(row.platform)
  }
  return [
    { value: null, label: t('admin.upstreams.compare.allPlatforms') },
    ...Array.from(platforms).sort().map((p) => ({ value: p, label: p }))
  ]
})

const formatPerMillion = (perToken: number | null | undefined): string =>
  perToken == null ? t('admin.upstreams.compare.noData') : `$${(perToken * 1e6).toFixed(4)}`

const load = async () => {
  loading.value = true
  try {
    result.value = await adminAPI.upstreams.compare({
      platform: platformFilter.value ?? undefined,
      search: search.value.trim() || undefined
    })
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.messages.loadFailed'))
  } finally {
    loading.value = false
  }
}

const onSearchInput = () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => void load(), 300)
}

onMounted(load)
</script>
