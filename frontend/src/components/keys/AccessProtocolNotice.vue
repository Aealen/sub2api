<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  baseUrl: string
}>()

const { t } = useI18n()

const COLLAPSE_STORAGE_KEY = 'keysAccessNoticeCollapsed'

// 站点根地址（容忍配置了 /v1 结尾或尾斜杠），与端点条口径一致
const baseRoot = computed(() => props.baseUrl.replace(/\/v1\/?$/, '').replace(/\/+$/, ''))
const baseUrlV1 = computed(() => (baseRoot.value.endsWith('/v1') ? baseRoot.value : `${baseRoot.value}/v1`))

const collapsed = ref(globalThis.localStorage?.getItem(COLLAPSE_STORAGE_KEY) === '1')

function toggleCollapsed() {
  collapsed.value = !collapsed.value
  try {
    globalThis.localStorage?.setItem(COLLAPSE_STORAGE_KEY, collapsed.value ? '1' : '0')
  } catch {
    // 无 localStorage 时折叠状态不记忆，不影响功能
  }
}

const protocolRows = computed(() => [
  { key: 'claude', title: t('keys.accessNotice.claudeTitle'), text: t('keys.accessNotice.claude', { baseUrl: baseRoot.value }) },
  { key: 'gpt', title: t('keys.accessNotice.gptTitle'), text: t('keys.accessNotice.gpt', { baseUrlV1: baseUrlV1.value }) },
  { key: 'grok', title: t('keys.accessNotice.grokTitle'), text: t('keys.accessNotice.grok') },
  { key: 'domestic', title: t('keys.accessNotice.domesticTitle'), text: t('keys.accessNotice.domestic') },
])

const billingItems = computed(() => [
  t('keys.accessNotice.billing1'),
  t('keys.accessNotice.billing2'),
  t('keys.accessNotice.billing3'),
  t('keys.accessNotice.billing4'),
  t('keys.accessNotice.billing5'),
])
</script>

<template>
  <div
    class="rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800"
    data-testid="keys-access-notice"
  >
    <button
      type="button"
      class="flex w-full items-center gap-2.5 px-4 py-2.5 text-left"
      :aria-expanded="!collapsed"
      data-testid="keys-access-notice-toggle"
      @click="toggleCollapsed"
    >
      <span
        class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-primary-100 text-primary-600 dark:bg-primary-900/40 dark:text-primary-300"
      >
        <svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </span>
      <span class="min-w-0 flex-1 text-sm font-semibold text-gray-900 dark:text-gray-100">
        {{ t('keys.accessNotice.title') }}
      </span>
      <svg
        class="h-4 w-4 shrink-0 text-gray-400 transition-transform duration-200 dark:text-gray-500"
        :class="collapsed ? '' : 'rotate-180'"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="2"
      >
        <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
      </svg>
    </button>

    <div v-show="!collapsed" class="px-4 pb-3.5 pl-11">
      <p class="text-xs leading-5 text-gray-500 dark:text-gray-400">
        {{ t('keys.accessNotice.cacheIntro') }}
      </p>

      <div class="mt-3 grid gap-2 sm:grid-cols-2">
        <div
          v-for="row in protocolRows"
          :key="row.key"
          class="rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-900/60"
        >
          <p class="text-xs font-semibold text-gray-800 dark:text-gray-200">{{ row.title }}</p>
          <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">
            {{ row.text }}
          </p>
        </div>
      </div>

      <div class="mt-3 rounded-lg border border-dashed border-primary-200 bg-primary-50/60 px-3 py-2 dark:border-primary-800 dark:bg-primary-900/20">
        <p class="text-xs font-semibold text-primary-700 dark:text-primary-300">
          {{ t('keys.accessNotice.baseUrlRuleTitle') }}
        </p>
        <p class="mt-1 text-xs leading-5 text-primary-600/90 dark:text-primary-400/90">
          {{ t('keys.accessNotice.baseUrlRule') }}
        </p>
      </div>

      <div class="mt-3">
        <p class="text-xs font-semibold text-gray-800 dark:text-gray-200">
          {{ t('keys.accessNotice.billingTitle') }}
        </p>
        <ul class="mt-1 space-y-0.5">
          <li
            v-for="(item, index) in billingItems"
            :key="index"
            class="flex gap-1.5 text-xs leading-5 text-gray-500 dark:text-gray-400"
          >
            <span class="mt-2 h-1 w-1 shrink-0 rounded-full bg-gray-300 dark:bg-dark-500"></span>
            <span>{{ item }}</span>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>