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
                :placeholder="t('admin.upstreams.compare.searchPlaceholder')"
                class="input pl-10"
                data-testid="upstream-compare-search"
                @input="debouncedLoad"
              />
            </div>
            <Select
              v-model="platformFilter"
              :options="platformOptions"
              :placeholder="t('admin.upstreams.compare.allPlatforms')"
              class="w-44"
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

        <div v-else class="overflow-x-auto">
          <table class="min-w-full border-separate border-spacing-0 text-sm" data-testid="upstream-compare-table">
            <thead>
              <tr class="text-left text-xs uppercase text-gray-500 dark:text-gray-400">
                <th class="sticky left-0 z-10 bg-white px-3 py-2 dark:bg-dark-800">{{ t('admin.upstreams.compare.model') }}</th>
                <th class="px-3 py-2">{{ t('admin.upstreams.compare.official') }}</th>
                <th
                  v-for="upstream in result.upstreams"
                  :key="upstream.id"
                  class="whitespace-nowrap px-3 py-2"
                >
                  {{ upstream.name }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="row in result.rows"
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
                  v-for="upstream in result.upstreams"
                  :key="upstream.id"
                  class="whitespace-nowrap px-3 py-2"
                >
                  <template v-if="cellFor(row, upstream.id)">
                    <div
                      class="inline-flex flex-col rounded-md px-2 py-1"
                      :class="row.cheapest_upstream_id === upstream.id
                        ? 'bg-green-50 ring-1 ring-green-300 dark:bg-green-900/20 dark:ring-green-700'
                        : ''"
                      :data-testid="`upstream-compare-cell-${upstream.id}`"
                    >
                      <span class="font-mono text-xs font-medium text-gray-900 dark:text-gray-100">
                        {{ formatPerMillion(cellFor(row, upstream.id)?.input_price_per_token) }}
                      </span>
                      <span class="text-[10px] text-gray-400">
                        <template v-if="cellFor(row, upstream.id)?.rate_multiplier != null">
                          ×{{ cellFor(row, upstream.id)?.rate_multiplier }}
                        </template>
                        <span
                          class="ml-1 rounded px-1"
                          :class="cellFor(row, upstream.id)?.source === 'ledger'
                            ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
                            : 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300'"
                        >
                          {{ cellFor(row, upstream.id)?.source === 'ledger'
                            ? t('admin.upstreams.compare.sourceLedger')
                            : t('admin.upstreams.compare.sourceDerived') }}
                        </span>
                      </span>
                      <span
                        v-if="cellFor(row, upstream.id)?.account_name"
                        class="text-[10px] text-gray-400"
                      >
                        {{ cellFor(row, upstream.id)?.account_name }}
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
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { UpstreamCompareResult, UpstreamCompareRow } from '@/api/admin/upstreams'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const result = ref<UpstreamCompareResult | null>(null)
const platformFilter = ref<string | null>(null)
const search = ref('')
let searchTimer: ReturnType<typeof setTimeout> | null = null

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

const cellFor = (row: UpstreamCompareRow, upstreamId: number) =>
  row.cells?.find((cell) => cell.upstream_id === upstreamId) ?? null

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

const debouncedLoad = () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => void load(), 300)
}

onMounted(load)
</script>
