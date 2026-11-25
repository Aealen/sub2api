<template>
  <BaseDialog
    :show="show"
    :title="t('admin.upstreams.overview.rateDialog.title')"
    width="normal"
    @close="close"
  >
    <div class="space-y-4">
      <!-- 只读上下文：让用户在改倍率前确认改的是哪一条上游分组 -->
      <div class="space-y-2 rounded-lg bg-gray-50 px-3 py-2.5 text-sm dark:bg-dark-700/40">
        <div class="flex items-baseline gap-2">
          <span class="shrink-0 text-gray-500 dark:text-gray-400">
            {{ t('admin.upstreams.overview.rateDialog.upstream') }}
          </span>
          <span class="font-medium text-gray-900 dark:text-white">{{ upstreamName }}</span>
        </div>
        <div class="flex items-baseline gap-2">
          <span class="shrink-0 text-gray-500 dark:text-gray-400">
            {{ t('admin.upstreams.overview.rateDialog.upstreamGroup') }}
          </span>
          <span class="font-medium text-gray-900 dark:text-white">{{ upstreamGroupName }}</span>
        </div>
      </div>

      <div>
        <label class="input-label" for="upstream-group-rate-input">
          {{ t('admin.upstreams.overview.rateDialog.rate') }}
        </label>
        <input
          id="upstream-group-rate-input"
          v-model="rateInput"
          type="number"
          step="0.0001"
          min="0"
          class="input mt-1 w-full"
          :placeholder="t('admin.upstreams.overview.rateDialog.ratePlaceholder')"
          data-testid="upstream-group-rate-input"
          @keyup.enter="handleSave"
        />
        <p v-if="rateInvalid" class="input-error-text mt-1.5">
          {{ t('admin.upstreams.overview.rateDialog.invalidRate') }}
        </p>
      </div>

      <p class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.upstreams.overview.rateDialog.hint') }}
      </p>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="close">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="!canSubmit"
          data-testid="upstream-group-rate-save"
          @click="handleSave"
        >
          {{ t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { UpstreamGroup } from '@/api/admin/upstreams'

const props = withDefaults(defineProps<{
  show: boolean
  upstreamId: number
  upstreamGroupId: number
  upstreamName: string
  upstreamGroupName: string
  rateMultiplier: number | null
}>(), {
  rateMultiplier: null
})

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'saved'): void
  (e: 'refresh'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

/** 输入框保留原始字符串：空串与 0 必须区分开，否则空值会被当成合法倍率 */
const rateInput = ref('')
const saving = ref(false)

const parsedRate = computed<number | null>(() => {
  const raw = rateInput.value.trim()
  if (raw === '') return null
  const value = Number(raw)
  if (!Number.isFinite(value) || value < 0) return null
  return value
})

/** 已输入但不合法（例如负数）：仅在这种情况下提示，空输入只禁用保存 */
const rateInvalid = computed(() => rateInput.value.trim() !== '' && parsedRate.value == null)

const canSubmit = computed(() => !saving.value && parsedRate.value != null)

const reset = () => {
  rateInput.value = props.rateMultiplier == null ? '' : String(props.rateMultiplier)
  saving.value = false
}

watch(
  () => props.show,
  (show) => {
    if (show) reset()
  },
  { immediate: true }
)

const close = () => emit('update:show', false)

const handleSave = async () => {
  const value = parsedRate.value
  if (!canSubmit.value || value == null) return
  saving.value = true
  try {
    // upsertGroup 是全字段覆盖：只传倍率会把峰时倍率/时段/备注/来源清空，
    // 因此必须先取该上游分组的现值合并后再提交。
    const detail = await adminAPI.upstreams.getById(props.upstreamId)
    const current = detail.groups.find((group) => group.id === props.upstreamGroupId)
    if (!current) {
      appStore.showError(t('admin.upstreams.overview.rateDialog.saveFailed'))
      return
    }
    await adminAPI.upstreams.upsertGroup(props.upstreamId, {
      id: props.upstreamGroupId,
      name: current.name,
      rate_multiplier: value,
      peak_rate_multiplier: current.peak_rate_multiplier,
      peak_start: current.peak_start,
      peak_end: current.peak_end,
      source: current.source as UpstreamGroup['source'],
      note: current.note
    })
    appStore.showSuccess(t('admin.upstreams.overview.rateDialog.saved'))
    emit('saved')
    emit('refresh')
    close()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.overview.rateDialog.saveFailed'))
  } finally {
    saving.value = false
  }
}
</script>
