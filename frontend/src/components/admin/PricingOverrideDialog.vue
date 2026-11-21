<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
      @click.self="close"
    >
      <div class="flex max-h-[85vh] w-full max-w-3xl flex-col rounded-2xl border border-gray-100 bg-white shadow-xl dark:border-dark-700 dark:bg-dark-800">
        <!-- 头部 -->
        <div class="flex items-center justify-between border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <div class="min-w-0">
            <h3 class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('admin.channels.pricingOverride.title') }}
            </h3>
            <p class="mt-0.5 truncate font-mono text-xs text-gray-400 dark:text-dark-500">
              {{ configPath || t('admin.channels.pricingOverride.notConfigured') }}
            </p>
          </div>
          <button
            type="button"
            class="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700 dark:hover:text-gray-200"
            :aria-label="t('common.close')"
            @click="close"
          >
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- 编辑区 -->
        <div class="flex-1 overflow-y-auto px-5 py-4">
          <p class="mb-3 text-xs leading-relaxed text-gray-500 dark:text-dark-400">
            {{ t('admin.channels.pricingOverride.hint') }}
          </p>
          <textarea
            v-model="content"
            spellcheck="false"
            class="h-72 w-full resize-y rounded-xl border border-gray-200 bg-gray-900 p-4 font-mono text-xs leading-relaxed text-gray-100 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:border-dark-600 dark:bg-dark-950"
            :disabled="loading || saving"
          ></textarea>
          <p v-if="jsonError" class="mt-2 text-xs text-red-500">{{ jsonError }}</p>

          <div class="mt-3 rounded-lg bg-amber-50 px-3 py-2 text-xs leading-relaxed text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
            {{ t('admin.channels.pricingOverride.warning') }}
          </div>

          <p v-if="saveMessage" class="mt-3 text-xs" :class="saveOk ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-500'">
            {{ saveMessage }}
          </p>
        </div>

        <!-- 底部按钮 -->
        <div class="flex items-center justify-end gap-3 border-t border-gray-100 px-5 py-4 dark:border-dark-700">
          <button
            type="button"
            class="rounded-xl px-4 py-2 text-sm font-medium text-gray-600 transition-colors hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
            :disabled="saving"
            @click="close"
          >
            {{ t('common.cancel') }}
          </button>
          <button
            type="button"
            class="btn-primary rounded-xl px-5 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="loading || saving || !!jsonError"
            @click="save"
          >
            {{ saving ? t('admin.channels.pricingOverride.saving') : t('admin.channels.pricingOverride.save') }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getPricingOverride, updatePricingOverride } from '@/api/admin/pricingOverride'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ 'update:visible': [value: boolean] }>()
const { t } = useI18n()

const loading = ref(false)
const saving = ref(false)
const content = ref('')
const configPath = ref('')
const saveMessage = ref('')
const saveOk = ref(false)

// JSON 实时校验：空串视为未初始化（不阻塞保存前的查看），非空则必须为合法 JSON 对象
const jsonError = computed(() => {
  const raw = content.value.trim()
  if (!raw) return null
  try {
    const parsed = JSON.parse(raw)
    if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return t('admin.channels.pricingOverride.jsonObjectRequired')
    }
    return null
  } catch {
    return t('admin.channels.pricingOverride.jsonInvalid')
  }
})

watch(
  () => props.visible,
  async (v) => {
    if (!v) return
    saveMessage.value = ''
    loading.value = true
    try {
      const resp = await getPricingOverride()
      configPath.value = resp.configured ? resp.path : ''
      content.value = resp.content || ''
      if (!resp.configured) {
        saveMessage.value = t('admin.channels.pricingOverride.notConfiguredHint')
        saveOk.value = false
      }
    } catch {
      saveMessage.value = t('admin.channels.pricingOverride.loadFailed')
      saveOk.value = false
    } finally {
      loading.value = false
    }
  },
  { immediate: true }
)

function close() {
  emit('update:visible', false)
}

async function save() {
  if (jsonError.value) return
  saving.value = true
  saveMessage.value = ''
  try {
    const result = await updatePricingOverride(content.value.trim())
    if (result.reloaded) {
      saveOk.value = true
      saveMessage.value = t('admin.channels.pricingOverride.savedReloaded')
    } else {
      saveOk.value = true
      saveMessage.value = t('admin.channels.pricingOverride.savedReloadPending') + (result.warning ? ` (${result.warning})` : '')
    }
  } catch {
    saveOk.value = false
    saveMessage.value = t('admin.channels.pricingOverride.saveFailed')
  } finally {
    saving.value = false
  }
}
</script>
