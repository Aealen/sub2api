<template>
  <BaseDialog
    :show="show"
    :title="t('admin.upstreams.overview.dialog.title')"
    width="normal"
    @close="close"
  >
    <div class="space-y-4">
      <div>
        <label class="input-label">{{ t('admin.upstreams.overview.dialog.group') }}</label>
        <Select
          v-model="groupId"
          :options="groupOptions"
          :disabled="lockedGroupId != null"
          :placeholder="t('admin.upstreams.overview.dialog.groupPlaceholder')"
          class="mt-1 w-full"
          data-testid="group-link-dialog-group"
        />
      </div>

      <div>
        <label class="input-label">{{ t('admin.upstreams.overview.dialog.upstreamGroup') }}</label>
        <Select
          v-model="upstreamGroupId"
          :options="upstreamGroupOptions"
          :placeholder="t('admin.upstreams.overview.dialog.upstreamGroupPlaceholder')"
          class="mt-1 w-full"
          data-testid="group-link-dialog-upstream-group"
        />
        <p
          v-if="upstreamGroupOptions.length === 0"
          class="mt-1 text-xs text-amber-600 dark:text-amber-400"
          data-testid="group-link-dialog-upstream-group-empty"
        >
          {{ t('admin.upstreams.overview.dialog.upstreamGroupEmpty') }}
        </p>
      </div>

      <Input
        v-model="note"
        :label="t('admin.upstreams.overview.dialog.note')"
        :placeholder="t('admin.upstreams.overview.dialog.notePlaceholder')"
        data-testid="group-link-dialog-note"
      />
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
          data-testid="group-link-dialog-save"
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
import Select from '@/components/common/Select.vue'
import Input from '@/components/common/Input.vue'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { UpstreamGroupCatalog } from '@/api/admin/upstreams'

/** 本站分组选项：页面把 links 与 unlinked_groups 合并后传入 */
interface SiteGroupOption {
  group_id: number
  group_name: string
  platform: string
  group_rate_multiplier: number
}

/** 可选的「上游 × 上游分组」扁平行（用类型别名而非 interface，才能满足 Select 的 options 约束） */
type UpstreamGroupOption = {
  value: number
  label: string
}

const props = withDefaults(defineProps<{
  show: boolean
  groups: SiteGroupOption[]
  /** 上游分组全集：由概览接口的 upstream_groups 一次传入，不再逐个上游拉取 */
  upstreamGroups: UpstreamGroupCatalog[]
  /** 预先锁定的本站分组（从未关联分组进入时使用） */
  prefillGroupId?: number | null
}>(), {
  prefillGroupId: null
})

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'saved'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const saving = ref(false)
const groupId = ref<number | null>(null)
const upstreamGroupId = ref<number | null>(null)
const note = ref('')

/** 锁定的分组：预填后不允许再改，避免「以为是新建，实际改到别的分组」 */
const lockedGroupId = computed<number | null>(() => props.prefillGroupId ?? null)

const formatRate = (value: number | null | undefined): string => {
  if (value == null || !Number.isFinite(value)) return '—'
  return `×${Number(value.toFixed(4))}`
}

const groupOptions = computed(() =>
  props.groups.map((group) => ({
    value: group.group_id,
    label: [group.group_name, group.platform, formatRate(group.group_rate_multiplier)]
      .filter((part) => Boolean(part))
      .join(' · ')
  }))
)

const upstreamGroupOptions = computed<UpstreamGroupOption[]>(() =>
  props.upstreamGroups.map((group) => ({
    value: group.id,
    label: `${group.upstream_name} · ${group.name} ${formatRate(group.rate_multiplier)}`
  }))
)

const canSubmit = computed(
  () => !saving.value && groupId.value != null && upstreamGroupId.value != null
)

const reset = () => {
  groupId.value = props.prefillGroupId ?? null
  upstreamGroupId.value = null
  note.value = ''
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
  if (!canSubmit.value || groupId.value == null || upstreamGroupId.value == null) return
  saving.value = true
  try {
    await adminAPI.upstreams.upsertGroupLink({
      group_id: groupId.value,
      upstream_group_id: upstreamGroupId.value,
      note: note.value.trim() || undefined
    })
    appStore.showSuccess(t('admin.upstreams.overview.dialog.saved'))
    emit('saved')
    close()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.overview.dialog.saveFailed'))
  } finally {
    saving.value = false
  }
}
</script>
