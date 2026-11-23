<template>
  <BaseDialog :show="show" :title="isEdit ? t('admin.upstreams.form.editTitle') : t('admin.upstreams.form.createTitle')" @close="close">
    <div class="space-y-4">
      <div>
        <label class="input-label">{{ t('admin.upstreams.form.name') }}</label>
        <input
          v-model="form.name"
          type="text"
          class="input mt-1 w-full"
          :placeholder="t('admin.upstreams.form.namePlaceholder')"
          data-testid="upstream-form-name"
        />
      </div>

      <div>
        <label class="input-label">{{ t('admin.upstreams.form.baseUrl') }}</label>
        <input
          v-model="form.base_url"
          type="text"
          class="input mt-1 w-full"
          :placeholder="t('admin.upstreams.form.baseUrlPlaceholder')"
          data-testid="upstream-form-base-url"
        />
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('admin.upstreams.form.baseUrlHint') }}</p>
      </div>

      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.upstreams.form.website') }}</label>
          <input v-model="form.website" type="text" class="input mt-1 w-full" data-testid="upstream-form-website" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.upstreams.form.contact') }}</label>
          <input v-model="form.contact" type="text" class="input mt-1 w-full" data-testid="upstream-form-contact" />
        </div>
      </div>

      <div>
        <label class="input-label">{{ t('admin.upstreams.form.status') }}</label>
        <Select
          v-model="form.status"
          :options="statusOptions"
          class="mt-1 w-full sm:w-48"
          data-testid="upstream-form-status"
        />
      </div>

      <div>
        <label class="input-label">{{ t('admin.upstreams.form.note') }}</label>
        <textarea v-model="form.note" rows="3" class="input mt-1 w-full" data-testid="upstream-form-note"></textarea>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="close">{{ t('common.cancel') }}</button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="saving || !form.name.trim()"
          data-testid="upstream-form-save"
          @click="handleSave"
        >
          {{ t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { Upstream } from '@/api/admin/upstreams'

const props = defineProps<{
  show: boolean
  upstream?: Upstream | null
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'saved', upstream: Upstream): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const saving = ref(false)

const isEdit = computed(() => Boolean(props.upstream?.id))

const form = reactive({
  name: '',
  base_url: '',
  website: '',
  contact: '',
  note: '',
  status: 'active' as 'active' | 'inactive'
})

const statusOptions = computed(() => [
  { value: 'active', label: t('admin.upstreams.status.active') },
  { value: 'inactive', label: t('admin.upstreams.status.inactive') }
])

const reset = (upstream?: Upstream | null) => {
  form.name = upstream?.name ?? ''
  form.base_url = upstream?.base_url ?? ''
  form.website = upstream?.website ?? ''
  form.contact = upstream?.contact ?? ''
  form.note = upstream?.note ?? ''
  form.status = upstream?.status === 'inactive' ? 'inactive' : 'active'
}

watch(
  () => [props.show, props.upstream] as const,
  ([show]) => {
    if (show) reset(props.upstream)
  },
  { immediate: true }
)

const close = () => emit('update:show', false)

const handleSave = async () => {
  if (saving.value || !form.name.trim()) return
  saving.value = true
  try {
    const payload = {
      name: form.name.trim(),
      base_url: form.base_url.trim(),
      website: form.website.trim(),
      contact: form.contact.trim(),
      note: form.note.trim(),
      status: form.status
    }
    const saved = props.upstream?.id
      ? await adminAPI.upstreams.update(props.upstream.id, payload)
      : await adminAPI.upstreams.create(payload)
    appStore.showSuccess(isEdit.value ? t('admin.upstreams.messages.saved') : t('admin.upstreams.messages.created'))
    emit('saved', saved)
    close()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.messages.saveFailed'))
  } finally {
    saving.value = false
  }
}
</script>
