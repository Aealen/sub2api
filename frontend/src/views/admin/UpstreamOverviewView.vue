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
                :placeholder="t('admin.upstreams.overview.searchPlaceholder')"
                class="input pl-10"
                data-testid="upstream-overview-search"
              />
            </div>

            <div
              class="inline-flex overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800"
              role="group"
            >
              <button
                v-for="option in viewOptions"
                :key="option.value"
                type="button"
                class="px-3 py-2 text-sm transition-colors"
                :class="view === option.value
                  ? 'bg-primary-500 text-white'
                  : 'text-gray-600 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-700'"
                :aria-pressed="view === option.value"
                :data-testid="`upstream-overview-view-${option.value}`"
                @click="view = option.value"
              >
                {{ option.label }}
              </button>
            </div>

            <div
              class="inline-flex overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800"
              role="group"
            >
              <button
                v-for="days in windowOptions"
                :key="days"
                type="button"
                class="px-3 py-2 text-sm transition-colors"
                :class="windowDays === days
                  ? 'bg-primary-500 text-white'
                  : 'text-gray-600 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-dark-700'"
                :aria-pressed="windowDays === days"
                :data-testid="`upstream-overview-window-${days}`"
                @click="windowDays = days"
              >
                {{ t('admin.upstreams.overview.windowDays', { days }) }}
              </button>
            </div>
          </div>

          <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-3 lg:w-auto">
            <button
              class="btn btn-primary"
              data-testid="upstream-overview-link-create"
              @click="openLinkDialog(null)"
            >
              <Icon name="link" size="md" class="mr-2" />
              {{ t('admin.upstreams.overview.link.create') }}
            </button>
            <button class="btn btn-secondary" :disabled="loading" :title="t('common.refresh')" @click="load">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
        <p class="mt-3 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.upstreams.overview.description') }}
        </p>
      </template>

      <template #table>
        <div class="flex h-full min-h-0 flex-col">
          <!-- 不设 max-h：展开余额卡后自然撑开，高度压力交给下方自带滚动的表格区 -->
          <div class="shrink-0 border-b border-gray-100 px-4 py-3 dark:border-dark-700">
            <UpstreamBalanceBar :upstreams="result?.upstreams ?? []" :loading="loading" />
          </div>

          <UpstreamLinkTable
            :result="result"
            :view="view"
            :window-days="windowDays"
            :keyword="search"
            :loading="loading"
            @refresh="load"
            @edit-rate="onEditRate"
          />

          <div
            v-if="unlinkedGroups.length > 0"
            class="shrink-0 border-t border-amber-200 bg-amber-50/60 px-4 py-3 dark:border-amber-500/30 dark:bg-amber-900/10"
            data-testid="upstream-overview-unlinked"
          >
            <p class="flex items-center gap-1.5 text-sm font-medium text-amber-800 dark:text-amber-300">
              <Icon name="exclamationTriangle" size="sm" class="shrink-0 text-amber-500" />
              {{ t('admin.upstreams.overview.unlinked.title') }}
            </p>
            <p class="mt-1 text-xs text-amber-700/80 dark:text-amber-300/70">
              {{ t('admin.upstreams.overview.unlinked.hint') }}
            </p>
            <div class="mt-2 flex max-h-24 flex-wrap gap-2 overflow-y-auto">
              <span
                v-for="group in unlinkedGroups"
                :key="group.group_id"
                class="inline-flex items-center gap-1.5 rounded-full border border-amber-300 bg-white/80 px-2.5 py-1 text-xs text-amber-800 dark:border-amber-500/40 dark:bg-dark-800/60 dark:text-amber-200"
              >
                <span class="font-medium">{{ group.group_name }}</span>
                <span v-if="group.platform" class="text-amber-600/70 dark:text-amber-300/60">
                  {{ group.platform }}
                </span>
                <span class="font-mono">{{ formatRate(group.group_rate_multiplier) }}</span>
                <button
                  type="button"
                  class="rounded-full bg-primary-600 px-2 py-0.5 text-[11px] font-medium text-white transition-colors hover:bg-primary-700"
                  :data-testid="`upstream-overview-link-unlinked-${group.group_id}`"
                  @click="openLinkDialog(group.group_id)"
                >
                  {{ t('admin.upstreams.overview.link.create') }}
                </button>
              </span>
            </div>
          </div>
        </div>
      </template>
    </TablePageLayout>

    <GroupLinkDialog
      v-model:show="linkDialogVisible"
      :groups="siteGroupOptions"
      :upstream-groups="result?.upstream_groups ?? []"
      :prefill-group-id="linkPrefillGroupId"
      @saved="onLinkSaved"
    />

    <UpstreamGroupRateDialog
      v-model:show="rateDialogVisible"
      :upstream-id="rateTarget.upstreamId"
      :upstream-group-id="rateTarget.upstreamGroupId"
      :upstream-name="rateTarget.upstreamName"
      :upstream-group-name="rateTarget.upstreamGroupName"
      :rate-multiplier="rateTarget.rateMultiplier"
      @refresh="load"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import UpstreamBalanceBar from '@/components/admin/upstream/UpstreamBalanceBar.vue'
import UpstreamLinkTable from '@/components/admin/upstream/UpstreamLinkTable.vue'
import GroupLinkDialog from '@/components/admin/upstream/GroupLinkDialog.vue'
import UpstreamGroupRateDialog from '@/components/admin/upstream/UpstreamGroupRateDialog.vue'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { UpstreamOverviewResult } from '@/api/admin/upstreams'

type OverviewView = 'group' | 'upstream'

/** 本站分组选项：由已关联分组与未关联分组合并而来 */
interface SiteGroupOption {
  group_id: number
  group_name: string
  platform: string
  group_rate_multiplier: number
}

/** 就地编辑进价倍率的目标：由表格子行的 edit-rate 事件带入 */
interface RateTarget {
  upstreamId: number
  upstreamGroupId: number
  upstreamName: string
  upstreamGroupName: string
  rateMultiplier: number | null
}

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const result = ref<UpstreamOverviewResult | null>(null)
const search = ref('')
const view = ref<OverviewView>('group')
const windowDays = ref(7)
const linkDialogVisible = ref(false)
const linkPrefillGroupId = ref<number | null>(null)
const rateDialogVisible = ref(false)
const rateTarget = ref<RateTarget>({
  upstreamId: 0,
  upstreamGroupId: 0,
  upstreamName: '',
  upstreamGroupName: '',
  rateMultiplier: null
})

/** 后端未返回窗口时的兜底选项 */
const DEFAULT_WINDOWS = [7, 30]

const viewOptions = computed<Array<{ value: OverviewView; label: string }>>(() => [
  { value: 'group', label: t('admin.upstreams.overview.viewByGroup') },
  { value: 'upstream', label: t('admin.upstreams.overview.viewByUpstream') }
])

const windowOptions = computed<number[]>(() => {
  const windows = result.value?.windows ?? []
  return windows.length > 0 ? windows : DEFAULT_WINDOWS
})

const unlinkedGroups = computed(() => result.value?.unlinked_groups ?? [])

/** 本站分组全集：links 里已关联的分组 + unlinked_groups，按 group_id 去重 */
const siteGroupOptions = computed<SiteGroupOption[]>(() => {
  const groups = new Map<number, SiteGroupOption>()
  const remember = (group: SiteGroupOption) => {
    if (!groups.has(group.group_id)) groups.set(group.group_id, group)
  }
  for (const link of result.value?.links ?? []) {
    remember({
      group_id: link.group_id,
      group_name: link.group_name,
      platform: link.platform,
      group_rate_multiplier: link.group_rate_multiplier
    })
  }
  for (const group of unlinkedGroups.value) {
    remember({
      group_id: group.group_id,
      group_name: group.group_name,
      platform: group.platform,
      group_rate_multiplier: group.group_rate_multiplier
    })
  }
  return [...groups.values()].sort((a, b) => a.group_name.localeCompare(b.group_name))
})

const formatRate = (value: number | null | undefined): string => {
  if (value == null || !Number.isFinite(value)) return '—'
  return `×${Number(value.toFixed(4))}`
}

/** 打开「建立关联」对话框；传 groupId 时锁定该本站分组 */
const openLinkDialog = (groupId: number | null = null) => {
  linkPrefillGroupId.value = groupId
  linkDialogVisible.value = true
}

const onLinkSaved = async () => {
  await load()
}

/** 打开「编辑上游进价倍率」弹窗；保存后由弹窗的 refresh 事件回到 load 重新拉取概览 */
const onEditRate = (payload: RateTarget) => {
  rateTarget.value = payload
  rateDialogVisible.value = true
}

const load = async () => {
  loading.value = true
  try {
    const next = await adminAPI.upstreams.getOverview()
    result.value = next
    const windows = next.windows ?? []
    if (windows.length > 0 && !windows.includes(windowDays.value)) {
      windowDays.value = windows[0]
    }
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.overview.messages.loadFailed'))
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
