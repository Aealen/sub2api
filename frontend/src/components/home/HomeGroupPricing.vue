<template>
  <section class="mb-16">
    <div class="mb-8 text-center">
      <h2 class="text-2xl font-bold text-gray-900 dark:text-white md:text-3xl">
        {{ t('home.pricing.title') }}
      </h2>
      <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('home.pricing.subtitle') }}</p>
    </div>

    <div class="grid gap-5 md:grid-cols-3">
      <div
        v-for="group in showcaseGroups"
        :key="group.id"
        class="flex flex-col rounded-2xl border border-gray-100 bg-white/70 p-5 shadow-card backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/50"
      >
        <!-- 组头：名称 + 倍率徽章 -->
        <div class="mb-3 flex items-center justify-between gap-2">
          <h3 class="truncate text-base font-semibold text-gray-900 dark:text-white">{{ group.name }}</h3>
          <span
            class="flex-shrink-0 rounded-full bg-primary-100 px-2.5 py-0.5 text-xs font-semibold text-primary-600 dark:bg-primary-900/30 dark:text-primary-400"
          >
            {{ group.rate_multiplier }}x
          </span>
        </div>
        <p class="mb-4 text-xs text-gray-400 dark:text-dark-500">
          {{ group.models.length }} {{ t('home.pricing.models') }}
          <template v-if="group.description">
            · {{ group.description }}
          </template>
        </p>

        <!-- 代表模型价格 -->
        <div class="mb-4 flex-1 space-y-2.5">
          <div
            v-for="model in group._showcaseModels"
            :key="model.name"
            class="rounded-xl bg-gray-50 px-3 py-2.5 dark:bg-dark-900/40"
          >
            <p class="truncate font-mono text-xs font-medium text-gray-700 dark:text-dark-200">{{ model.name }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
              {{ t('home.pricing.input') }}
              <span class="font-semibold text-primary-500 dark:text-primary-400">{{ model.input }}</span>
              <span class="mx-1.5 text-gray-300 dark:text-dark-600">|</span>
              {{ t('home.pricing.output') }}
              <span class="font-semibold text-primary-500 dark:text-primary-400">{{ model.output }}</span>
              <span class="text-gray-400 dark:text-dark-500">/ 1M</span>
            </p>
          </div>
        </div>

        <router-link
          to="/model-plaza"
          class="mt-auto inline-flex items-center justify-center gap-1 rounded-xl border border-primary-200 px-4 py-2 text-sm font-medium text-primary-600 transition-colors hover:bg-primary-50 dark:border-primary-800 dark:text-primary-400 dark:hover:bg-primary-900/20"
        >
          {{ t('home.pricing.viewAll') }}
          <Icon name="arrowRight" size="sm" :stroke-width="2" />
        </router-link>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'

const props = defineProps<{ groups: ModelPlazaGroup[] }>()
const { t } = useI18n()

/** USD/token → 每 1M token 展示值（去尾零，最多 3 位小数） */
function fmtPerMillion(price: number | null | undefined): string {
  if (price == null) return '—'
  const v = price * 1_000_000
  const s = v.toFixed(3).replace(/\.?0+$/, '')
  return '$' + (s === '' ? '0' : s)
}

/** 有完整输入/输出价的模型优先，取前 2 个作为代表 */
function pickShowcaseModels(models: PlazaModel[]) {
  return models
    .filter((m) => m.pricing?.input_price != null && m.pricing?.output_price != null)
    .slice(0, 2)
    .map((m) => ({
      name: m.name,
      input: fmtPerMillion(m.pricing?.input_price),
      output: fmtPerMillion(m.pricing?.output_price)
    }))
}

/** 标准(非订阅)分组按模型数降序取前 3 个，附上代表模型展示数据 */
const showcaseGroups = computed(() => {
  return props.groups
    .filter((g) => g.subscription_type !== 'subscription')
    .sort((a, b) => b.models.length - a.models.length)
    .slice(0, 3)
    .map((g) => ({ ...g, _showcaseModels: pickShowcaseModels(g.models) }))
    .filter((g) => g._showcaseModels.length > 0)
})
</script>
