<template>
  <!-- 空数据（模型广场未启用/无渠道）时整条隐藏，由父级控制 -->
  <div class="mb-12 grid grid-cols-3 gap-4 sm:gap-6">
    <div
      v-for="stat in stats"
      :key="stat.label"
      class="rounded-2xl border border-gray-100 bg-white/70 px-4 py-6 text-center shadow-card backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/50"
    >
      <p class="text-3xl font-bold text-primary-500 dark:text-primary-400 md:text-4xl">{{ stat.value }}</p>
      <p class="mt-1.5 text-sm text-gray-500 dark:text-dark-400">{{ stat.label }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelPlazaGroup } from '@/api/modelPlaza'

const props = defineProps<{ groups: ModelPlazaGroup[] }>()
const { t } = useI18n()

const totalModels = computed(() => props.groups.reduce((n, g) => n + g.models.length, 0))
const totalGroups = computed(() => props.groups.length)
const totalPlatforms = computed(() => new Set(props.groups.map((g) => g.platform)).size)

const stats = computed(() => [
  { value: totalModels.value, label: t('home.stats.models') },
  { value: totalGroups.value, label: t('home.stats.groups') },
  { value: totalPlatforms.value, label: t('home.stats.platforms') }
])
</script>
