<template>
  <BaseDialog :show="show" :title="t('admin.upstreams.unassigned.title')" width="wide" @close="close">
    <div class="space-y-4">
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.upstreams.unassigned.hint') }}</p>

      <div v-if="loading" class="flex items-center justify-center py-8">
        <LoadingSpinner />
      </div>

      <p v-else-if="hosts.length === 0" class="py-6 text-center text-sm text-gray-400">
        {{ t('admin.upstreams.unassigned.empty') }}
      </p>

      <table v-else class="min-w-full text-sm">
        <thead class="text-left text-xs uppercase text-gray-500 dark:text-gray-400">
          <tr>
            <th class="w-10 px-3 py-2"></th>
            <th class="px-3 py-2">{{ t('admin.upstreams.unassigned.host') }}</th>
            <th class="px-3 py-2">{{ t('admin.upstreams.unassigned.accounts') }}</th>
            <th class="px-3 py-2">{{ t('admin.upstreams.unassigned.sample') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-dark-600">
          <tr v-for="host in hosts" :key="host.host">
            <td class="px-3 py-2">
              <input
                type="checkbox"
                :value="host.host"
                :checked="selectedHosts.includes(host.host)"
                class="h-4 w-4 rounded border-gray-300 text-primary-600"
                @change="toggleHost(host.host)"
              />
            </td>
            <td class="px-3 py-2 font-mono text-xs text-gray-900 dark:text-white">{{ host.host || '—' }}</td>
            <td class="px-3 py-2 text-gray-600 dark:text-gray-300">{{ host.account_count }}</td>
            <td class="px-3 py-2 text-gray-500 dark:text-gray-400">{{ host.sample_names.join(', ') }}</td>
          </tr>
        </tbody>
      </table>

      <div v-if="hosts.length > 0" class="flex flex-wrap items-end gap-3 border-t border-gray-100 pt-3 dark:border-dark-600">
        <div class="min-w-[220px] flex-1">
          <label class="input-label">{{ t('admin.upstreams.unassigned.target') }}</label>
          <Select
            v-model="targetId"
            :options="upstreamOptions"
            :placeholder="t('admin.upstreams.unassigned.target')"
            class="mt-1 w-full"
            data-testid="unassigned-target-select"
          />
        </div>
        <button class="btn btn-secondary" @click="createTarget">
          {{ t('admin.upstreams.unassigned.createNew') }}
        </button>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="close">{{ t('common.cancel') }}</button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="saving || !canAssign"
          data-testid="unassigned-assign"
          @click="assign"
        >
          {{ t('admin.upstreams.unassigned.assign') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <UpstreamFormDialog v-model:show="createVisible" :upstream="null" @saved="onCreated" />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Select from '@/components/common/Select.vue'
import UpstreamFormDialog from './UpstreamFormDialog.vue'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { UnassignedAccountHost, Upstream, UpstreamOverview } from '@/api/admin/upstreams'

const props = defineProps<{
  show: boolean
  upstreams: UpstreamOverview[]
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'assigned'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const hosts = ref<UnassignedAccountHost[]>([])
const selectedHosts = ref<string[]>([])
const targetId = ref<number | null>(null)
const createVisible = ref(false)

const upstreamOptions = computed(() =>
  props.upstreams.map((u) => ({ value: u.id, label: u.name }))
)

const canAssign = computed(
  () => targetId.value != null && selectedHosts.value.length > 0
)

const toggleHost = (host: string) => {
  if (selectedHosts.value.includes(host)) {
    selectedHosts.value = selectedHosts.value.filter((h) => h !== host)
  } else {
    selectedHosts.value = [...selectedHosts.value, host]
  }
}

const load = async () => {
  loading.value = true
  try {
    const result = await adminAPI.upstreams.unassignedAccounts()
    hosts.value = result.items
    selectedHosts.value = []
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.messages.loadFailed'))
  } finally {
    loading.value = false
  }
}

watch(
  () => props.show,
  (show) => {
    if (show) void load()
  },
  { immediate: true }
)

const close = () => emit('update:show', false)

const createTarget = () => {
  createVisible.value = true
}

const onCreated = (upstream: Upstream) => {
  targetId.value = upstream.id
  createVisible.value = false
}

const assign = async () => {
  if (!canAssign.value || targetId.value == null || saving.value) return
  const accountIds = hosts.value
    .filter((h) => selectedHosts.value.includes(h.host))
    .flatMap((h) => h.account_ids)
  if (accountIds.length === 0) return
  saving.value = true
  try {
    await adminAPI.upstreams.assignAccounts(targetId.value, accountIds)
    appStore.showSuccess(t('admin.upstreams.messages.assigned', { count: accountIds.length }))
    emit('assigned')
    close()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.messages.assignFailed'))
  } finally {
    saving.value = false
  }
}
</script>
