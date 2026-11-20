<template>
  <section class="mb-16">
    <div class="mb-8 text-center">
      <h2 class="text-2xl font-bold text-gray-900 dark:text-white md:text-3xl">
        {{ t('home.guide.title') }}
      </h2>
      <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('home.guide.subtitle') }}</p>
      <p class="mt-3 inline-flex items-center gap-1.5 rounded-lg bg-gray-100 px-3 py-1.5 font-mono text-xs text-gray-600 dark:bg-dark-800/60 dark:text-dark-300">
        <Icon name="key" size="sm" />
        {{ baseUrl }}
      </p>
    </div>

    <div class="grid gap-5 md:grid-cols-3">
      <div
        v-for="ep in endpoints"
        :key="ep.title"
        class="rounded-2xl border border-gray-100 bg-white/70 p-5 shadow-card backdrop-blur-sm dark:border-dark-700/50 dark:bg-dark-800/50"
      >
        <h3 class="mb-1 text-sm font-semibold text-gray-900 dark:text-white">{{ ep.title }}</h3>
        <p class="mb-3 text-xs text-gray-400 dark:text-dark-500">{{ ep.desc }}</p>
        <div
          class="overflow-x-auto rounded-lg bg-gray-900 px-3 py-2.5 font-mono text-[11px] leading-relaxed text-gray-200 dark:bg-dark-950/70"
        >
          <span class="select-none text-emerald-400">POST</span>
          <span class="text-primary-300"> {{ ep.path }}</span>
        </div>
        <p class="mt-3 text-xs text-gray-500 dark:text-dark-400">{{ t('home.guide.hint') }}</p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const baseUrl = computed(() => window.location.origin)

const endpoints = computed(() => [
  { title: t('home.guide.anthropic'), desc: t('home.guide.anthropicDesc'), path: '/v1/messages' },
  { title: t('home.guide.openaiChat'), desc: t('home.guide.openaiChatDesc'), path: '/v1/chat/completions' },
  { title: t('home.guide.openaiResponses'), desc: t('home.guide.openaiResponsesDesc'), path: '/v1/responses' }
])
</script>
