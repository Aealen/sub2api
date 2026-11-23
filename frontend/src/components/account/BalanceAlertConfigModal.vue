<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.balance.configTitle')"
    width="wide"
    @close="handleClose"
  >
    <div v-if="loading" class="flex items-center justify-center py-8">
      <svg class="h-6 w-6 animate-spin text-gray-400" fill="none" viewBox="0 0 24 24">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
        <path
          class="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
        ></path>
      </svg>
    </div>

    <div v-else class="space-y-5">
      <!-- 同步频率 -->
      <div>
        <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.balance.syncInterval') }}
        </label>
        <select v-model.number="form.sync_interval_seconds" class="input w-full sm:w-64" data-testid="balance-sync-interval">
          <option v-for="opt in SYNC_INTERVAL_OPTIONS" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </option>
        </select>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ intervalHint }}
        </p>
      </div>

      <!-- 预警开关 -->
      <label class="flex items-center gap-2">
        <input v-model="form.alert_enabled" type="checkbox" class="h-4 w-4" data-testid="balance-alert-enabled" />
        <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('admin.accounts.balance.alertEnabled') }}</span>
      </label>

      <!-- 预警邮箱 -->
      <div>
        <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.balance.alertEmails') }}
        </label>
        <input
          v-model="alertEmailsInput"
          type="text"
          class="input w-full"
          :placeholder="t('admin.accounts.balance.alertEmailsPlaceholder')"
          data-testid="balance-alert-emails"
        />
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.balance.alertEmailsHint') }}
        </p>
      </div>

      <!-- 全局默认阈值 -->
      <div>
        <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.balance.globalThreshold') }}
        </label>
        <div class="flex items-center gap-2">
          <span class="text-sm text-gray-500 dark:text-gray-400">$</span>
          <input
            v-model.number="form.alert_threshold"
            type="number"
            min="0"
            step="0.01"
            class="input w-40"
            data-testid="balance-global-threshold"
          />
        </div>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.balance.globalThresholdHint') }}
        </p>
      </div>

      <!-- 专属阈值（覆盖全局） -->
      <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
        <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.balance.perAccountTitle') }}
        </p>
        <p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.balance.perAccountHint') }}
        </p>

        <!-- 添加行 -->
        <div class="mb-3 flex flex-wrap items-center gap-2">
          <select v-model="selectedAccountId" class="input w-56" data-testid="balance-per-account-select">
            <option :value="null" disabled>{{ t('admin.accounts.balance.selectAccount') }}</option>
            <option v-for="a in accountOptions" :key="a.id" :value="a.id">
              {{ a.name }}{{ a.upstream_balance != null ? ` ($${a.upstream_balance.toFixed(2)})` : '' }}
            </option>
          </select>
          <div class="flex items-center gap-1">
            <span class="text-sm text-gray-500 dark:text-gray-400">$</span>
            <input
              v-model.number="newThreshold"
              type="number"
              min="0"
              step="0.01"
              class="input w-28"
              :placeholder="t('admin.accounts.balance.thresholdPlaceholder')"
              data-testid="balance-per-account-threshold"
            />
          </div>
          <button
            type="button"
            class="btn btn-secondary px-3 py-1.5 text-sm"
            :disabled="selectedAccountId == null || newThreshold == null || newThreshold < 0"
            data-testid="balance-per-account-add"
            @click="addPerAccount"
          >
            {{ t('common.add') }}
          </button>
        </div>

        <!-- 已设置的账号 -->
        <div v-if="perAccountEntries.length === 0" class="text-sm text-gray-400 dark:text-dark-500">
          {{ t('admin.accounts.balance.noPerAccount') }}
        </div>
        <div v-else class="space-y-1.5">
          <div
            v-for="entry in perAccountEntries"
            :key="entry.id"
            class="flex items-center justify-between rounded-md bg-gray-50 px-3 py-1.5 dark:bg-dark-700/60"
            :data-testid="`balance-per-account-row-${entry.id}`"
          >
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ entry.name }}</span>
            <div class="flex items-center gap-2">
              <span class="font-mono text-sm text-gray-900 dark:text-gray-100">${{ entry.threshold.toFixed(2) }}</span>
              <button
                type="button"
                class="text-xs text-red-500 hover:text-red-600"
                :data-testid="`balance-per-account-remove-${entry.id}`"
                @click="removePerAccount(entry.id)"
              >
                {{ t('common.delete') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="btn btn-primary" :disabled="saving" data-testid="balance-config-save" @click="handleSave">
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
import accountsAPI, { type UpstreamBalanceConfig } from '@/api/admin/accounts'
import type { Account } from '@/types'

const props = defineProps<{
  show: boolean
  accounts: Account[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved', config?: UpstreamBalanceConfig): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const SYNC_INTERVAL_OPTIONS = [
  { value: 30, label: '30s' },
  { value: 60, label: '1 min' },
  { value: 300, label: '5 min' },
  { value: 600, label: '10 min' },
  { value: 1800, label: '30 min' }
]

const HIGH_FREQ_VALUES = new Set([30, 60])

const loading = ref(false)
const saving = ref(false)
const form = ref<UpstreamBalanceConfig>({
  sync_interval_seconds: 600,
  alert_enabled: true,
  alert_emails: [],
  alert_threshold: 10
})
const alertEmailsInput = ref('')
// 专属阈值待保存集：id -> {threshold, name}（保存时逐个调 API，未出现在集合里的账号清除专属设置）
const perAccountDraft = ref<Map<number, { threshold: number; name: string }>>(new Map())
const selectedAccountId = ref<number | null>(null)
const newThreshold = ref<number | null>(null)

const intervalHint = computed(() =>
  HIGH_FREQ_VALUES.has(form.value.sync_interval_seconds)
    ? t('admin.accounts.balance.intervalHighFreqHint')
    : t('admin.accounts.balance.intervalHint')
)

// 可选账号：apikey 型（有上游余额接口的转售站账号）
const accountOptions = computed(() => props.accounts.filter(a => a.type === 'apikey'))

// 待保存的专属阈值列表
const perAccountEntries = computed(() =>
  Array.from(perAccountDraft.value.entries())
    .map(([id, v]) => ({
      id,
      threshold: v.threshold,
      name: v.name || props.accounts.find(a => a.id === id)?.name || `#${id}`
    }))
    .sort((a, b) => a.name.localeCompare(b.name))
)

const loadConfig = async () => {
  loading.value = true
  try {
    const resp = await accountsAPI.getUpstreamBalanceConfig()
    form.value = {
      sync_interval_seconds: resp.sync_interval_seconds,
      alert_enabled: resp.alert_enabled,
      alert_emails: resp.alert_emails,
      alert_threshold: resp.alert_threshold
    }
    alertEmailsInput.value = resp.alert_emails.join(', ')
    // 预填专属阈值覆盖（后端全量返回，跨分页；名称来自后端）
    perAccountDraft.value = new Map(
      (resp.per_account_thresholds ?? []).map(o => [o.account_id, { threshold: o.threshold, name: o.account_name }])
    )
  } catch (error) {
    console.error('Failed to load upstream balance config:', error)
    appStore.showError(t('admin.accounts.balance.loadFailed'))
  } finally {
    loading.value = false
  }
}

watch(
  () => props.show,
  show => {
    if (show) void loadConfig()
  }
)

const addPerAccount = () => {
  if (selectedAccountId.value == null || newThreshold.value == null || newThreshold.value < 0) return
  const name = props.accounts.find(a => a.id === selectedAccountId.value)?.name ?? `#${selectedAccountId.value}`
  perAccountDraft.value.set(selectedAccountId.value, { threshold: newThreshold.value, name })
  // 触发 Map 响应式更新（新建 Map 副本）
  perAccountDraft.value = new Map(perAccountDraft.value)
  selectedAccountId.value = null
  newThreshold.value = null
}

const removePerAccount = (id: number) => {
  perAccountDraft.value.delete(id)
  perAccountDraft.value = new Map(perAccountDraft.value)
}

const handleSave = async () => {
  saving.value = true
  try {
    const emails = alertEmailsInput.value
      .split(/[,;，；]/)
      .map(e => e.trim())
      .filter(Boolean)
    if (emails.length === 0) {
      appStore.showError(t('admin.accounts.balance.emailsRequired'))
      return
    }
    const cfg: UpstreamBalanceConfig = {
      ...form.value,
      alert_emails: emails
    }
    await accountsAPI.updateUpstreamBalanceConfig(cfg)

    // 专属阈值：全部账号统一结算（草稿里有则设置；原先有但现在没有的清除）
    const previous = props.accounts.filter(a => a.balance_alert_threshold != null).map(a => a.id)
    const draftIds = new Set(perAccountDraft.value.keys())
    for (const [id, v] of perAccountDraft.value) {
      await accountsAPI.setAccountBalanceAlertThreshold(id, v.threshold)
    }
    for (const id of previous) {
      if (!draftIds.has(id)) {
        await accountsAPI.setAccountBalanceAlertThreshold(id, null)
      }
    }

    appStore.showSuccess(t('admin.accounts.balance.saved'))
    emit('saved', cfg)
    emit('close')
  } catch (error) {
    console.error('Failed to save upstream balance config:', error)
    appStore.showError(t('admin.accounts.balance.saveFailed'))
  } finally {
    saving.value = false
  }
}

const handleClose = () => {
  if (saving.value) return
  emit('close')
}
</script>
