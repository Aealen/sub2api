<template>
  <div v-if="loading && upstreams.length === 0" class="flex flex-wrap gap-3">
    <div
      v-for="i in 3"
      :key="i"
      class="h-28 min-w-[220px] flex-1 animate-pulse rounded-xl border border-gray-200 bg-gray-100 dark:border-dark-700 dark:bg-dark-700"
    ></div>
  </div>

  <div v-else-if="upstreams.length > 0" data-testid="upstream-balance-bar">
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-5">
      <div
        v-for="upstream in visibleUpstreams"
        :key="upstream.id"
        class="min-w-0 rounded-xl border p-3"
        :class="upstream.balance_alerting
          ? 'border-amber-400 bg-amber-50/60 ring-1 ring-amber-300 dark:border-amber-500/60 dark:bg-amber-900/10 dark:ring-amber-500/30'
          : 'border-gray-200 bg-gray-50/60 dark:border-dark-700 dark:bg-dark-800/60'"
        :data-testid="`upstream-balance-card-${upstream.id}`"
      >
      <div class="flex items-center justify-between gap-2">
        <span class="truncate text-sm font-semibold text-gray-900 dark:text-white">
          {{ upstream.name }}
        </span>
        <span
          class="inline-flex shrink-0 rounded-full px-2 py-0.5 text-[11px]"
          :class="upstream.status === 'active'
            ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
            : 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300'"
        >
          {{ t(`admin.upstreams.status.${upstream.status}`) }}
        </span>
      </div>

      <div class="mt-2 flex items-end justify-between gap-2">
        <div class="min-w-0">
          <div class="text-[11px] text-gray-500 dark:text-gray-400">
            {{ t('admin.upstreams.overview.balance.siteBalance') }}
          </div>
          <div
            class="font-mono text-lg font-semibold"
            :class="upstream.site_balance <= 0
              ? 'text-gray-400 dark:text-dark-400'
              : 'text-gray-900 dark:text-gray-100'"
          >
            {{ formatMoney(upstream.site_balance) }}
          </div>
        </div>
        <span
          v-if="upstream.balance_alerting"
          class="inline-flex shrink-0 items-center gap-1 rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-900/40 dark:text-amber-300"
          data-testid="upstream-balance-alert"
        >
          <Icon name="exclamationTriangle" size="xs" />
          {{ t('admin.upstreams.overview.balance.alert') }}
        </span>
      </div>

      <dl class="mt-2 grid grid-cols-3 gap-2 text-[11px] text-gray-500 dark:text-gray-400">
        <div>
          <dt>{{ t('admin.upstreams.overview.balance.sync') }}</dt>
          <dd class="mt-0.5 font-medium text-gray-700 dark:text-gray-200">
            {{ formatSyncTime(upstream.balance_updated_at) }}
          </dd>
        </div>
        <div>
          <dt>{{ t('admin.upstreams.overview.balance.accounts') }}</dt>
          <dd class="mt-0.5 font-mono font-medium text-gray-700 dark:text-gray-200">
            {{ upstream.account_active }}/{{ upstream.account_total }}
          </dd>
        </div>
        <div>
          <dt>{{ t('admin.upstreams.overview.balance.groups') }}</dt>
          <dd class="mt-0.5 font-mono font-medium text-gray-700 dark:text-gray-200">
            {{ upstream.group_count }}
          </dd>
        </div>
      </dl>
    </div>
    </div>

    <button
      v-if="upstreams.length > collapsedCount"
      type="button"
      class="mt-2 inline-flex items-center gap-1 text-xs font-medium text-primary-600 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
      data-testid="upstream-balance-expand"
      @click="expanded = !expanded"
    >
      <Icon :name="expanded ? 'chevronUp' : 'chevronDown'" size="xs" />
      {{ expanded ? t('admin.upstreams.overview.balance.collapse') : t('admin.upstreams.overview.balance.expand', { count: upstreams.length }) }}
    </button>
  </div>

  <p v-else class="text-xs text-gray-400 dark:text-dark-400">
    {{ t('admin.upstreams.overview.balance.empty') }}
  </p>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { UpstreamOverviewCard } from '@/api/admin/upstreams'

const props = withDefaults(defineProps<{
  upstreams: UpstreamOverviewCard[]
  loading?: boolean
}>(), {
  loading: false
})

const { t } = useI18n()

/** 卡片条默认只展示前 5 家，超出折叠；展开后 flex-wrap 自然换行向下追加 */
const collapsedCount = 5
const expanded = ref(false)
const visibleUpstreams = computed(() =>
  expanded.value ? props.upstreams : props.upstreams.slice(0, collapsedCount)
)

const formatMoney = (value: number | null | undefined): string =>
  value == null || !Number.isFinite(value) ? '—' : `$${value.toFixed(2)}`

/** 相对同步时间，口径与 UpstreamsView 保持一致 */
const formatSyncTime = (raw: string | null): string => {
  if (!raw) return '—'
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return '—'
  const diffMinutes = Math.floor((Date.now() - date.getTime()) / 60000)
  if (diffMinutes < 1) return t('common.justNow')
  if (diffMinutes < 60) return `${diffMinutes}m`
  if (diffMinutes < 1440) return `${Math.floor(diffMinutes / 60)}h`
  return date.toLocaleDateString()
}
</script>
