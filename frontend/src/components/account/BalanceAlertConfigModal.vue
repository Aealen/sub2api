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
        <label class="input-label">{{ t('admin.accounts.balance.syncInterval') }}</label>
        <Select
          v-model="form.sync_interval_seconds"
          :options="intervalOptions"
          class="mt-2 w-full sm:w-64"
          :aria-label="t('admin.accounts.balance.syncInterval')"
          data-testid="balance-sync-interval"
        />
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ intervalHint }}
        </p>
      </div>

      <!-- 预警开关 -->
      <div class="flex items-center justify-between gap-4">
        <label class="input-label mb-0">{{ t('admin.accounts.balance.alertEnabled') }}</label>
        <Toggle
          v-model="form.alert_enabled"
          data-testid="balance-alert-enabled"
          :aria-label="t('admin.accounts.balance.alertEnabled')"
        />
      </div>

      <!-- 预警邮箱 -->
      <div>
        <label class="input-label">{{ t('admin.accounts.balance.alertEmails') }}</label>
        <input
          v-model="alertEmailsInput"
          type="text"
          class="input mt-2 w-full"
          :placeholder="t('admin.accounts.balance.alertEmailsPlaceholder')"
          data-testid="balance-alert-emails"
        />
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.balance.alertEmailsHint') }}
        </p>
      </div>

      <!-- 全局默认阈值 -->
      <div>
        <label class="input-label">{{ t('admin.accounts.balance.globalThreshold') }}</label>
        <div class="mt-2 flex items-center gap-2">
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
        <p class="mb-1 text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.balance.perAccountTitle') }}
        </p>
        <p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.balance.perAccountHint') }}
        </p>

        <!-- 添加行 -->
        <div class="mb-3 flex flex-wrap items-center gap-2">
          <Select
            v-model="selectedAccountId"
            :options="accountSelectOptions"
            :placeholder="t('admin.accounts.balance.selectAccount')"
            :aria-label="t('admin.accounts.balance.selectAccount')"
            class="w-full sm:w-72"
            data-testid="balance-per-account-select"
          />
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
            :disabled="!canAddPerAccount"
            data-testid="balance-per-account-add"
            @click="addPerAccount"
          >
            {{ t('common.add') }}
          </button>
        </div>
        <p v-if="accountOptions.length === 0" class="mb-3 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.balance.noEligibleAccounts') }}
        </p>

        <!-- 已设置的账号 -->
        <div v-if="perAccountEntries.length === 0" class="text-sm text-gray-400 dark:text-dark-500">
          {{ t('admin.accounts.balance.noPerAccount') }}
        </div>
        <div v-else class="space-y-1.5">
          <div
            v-for="entry in perAccountEntries"
            :key="entry.id"
            class="flex items-center justify-between rounded-md bg-gray-50 px-3 py-1.5 dark:bg-dark-700/60"
          >
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ entry.name }}</span>
            <div class="flex items-center gap-2">
              <span class="font-mono text-sm text-gray-900 dark:text-gray-100">${{ entry.threshold.toFixed(2) }}</span>
              <button
                type="button"
                class="text-xs text-red-500 hover:text-red-600"
                :aria-label="t('common.delete')"
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
        <button type="button" class="btn btn-primary" :disabled="saving || loading" data-testid="balance-config-save" @click="handleSave">
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
import Toggle from '@/components/common/Toggle.vue'
import { useAppStore } from '@/stores/app'
import accountsAPI, {
  type UpstreamBalanceAccountOption,
  type UpstreamBalanceConfig
} from '@/api/admin/accounts'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved', config: UpstreamBalanceConfig): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

/** 同步间隔可选项（秒） */
const SYNC_INTERVAL_SECONDS = [30, 60, 300, 600, 1800]
/** 高频档：频繁请求上游接口可能触发风控，选中时展示提示 */
const HIGH_FREQ_VALUES = new Set<number>([30, 60])

const loading = ref(false)
const saving = ref(false)
const form = ref<UpstreamBalanceConfig>({
  sync_interval_seconds: 600,
  alert_enabled: true,
  alert_emails: [],
  alert_threshold: 10
})
const alertEmailsInput = ref('')
/** 可同步上游余额的全部账号（接口返回，不依赖列表页当前页） */
const accountOptions = ref<UpstreamBalanceAccountOption[]>([])
/** 专属阈值草稿：账号 id -> { threshold, name } */
const perAccountDraft = ref<Map<number, { threshold: number; name: string }>>(new Map())
/** 服务端当前的专属阈值（打开弹窗时载入）；保存时据此只提交有变化的账号 */
let loadedOverrides = new Map<number, number>()
const selectedAccountId = ref<number | null>(null)
const newThreshold = ref<number | null>(null)

const intervalLabel = (seconds: number): string =>
  seconds < 60
    ? t('admin.accounts.balance.intervalSeconds', { n: seconds })
    : t('admin.accounts.balance.intervalMinutes', { n: seconds / 60 })

const intervalOptions = computed(() =>
  SYNC_INTERVAL_SECONDS.map(seconds => ({ value: seconds, label: intervalLabel(seconds) }))
)

const intervalHint = computed(() =>
  HIGH_FREQ_VALUES.has(form.value.sync_interval_seconds)
    ? t('admin.accounts.balance.intervalHighFreqHint')
    : t('admin.accounts.balance.intervalHint')
)

const perAccountEntries = computed(() =>
  Array.from(perAccountDraft.value.entries())
    .map(([id, v]) => ({ id, threshold: v.threshold, name: v.name }))
    .sort((a, b) => a.name.localeCompare(b.name))
)

// 已设置专属阈值的账号在下拉里标注并禁用（修改需先删除再重新添加）
const accountSelectOptions = computed(() =>
  accountOptions.value.map(a => {
    const balance = a.upstream_balance != null ? ` ($${a.upstream_balance.toFixed(2)})` : ''
    const configured = perAccountDraft.value.has(a.id)
    return {
      value: a.id,
      label: configured ? `${a.name}${balance} — ${t('admin.accounts.balance.configured')}` : `${a.name}${balance}`,
      disabled: configured
    }
  })
)

const canAddPerAccount = computed(
  () => selectedAccountId.value != null && newThreshold.value != null && newThreshold.value >= 0
)

const loadConfig = async () => {
  loading.value = true
  try {
    const resp = await accountsAPI.getUpstreamBalanceConfig()
    form.value = { ...resp.config }
    alertEmailsInput.value = (resp.config.alert_emails ?? []).join(', ')
    accountOptions.value = resp.accounts ?? []
    const overrides = resp.per_account_thresholds ?? []
    loadedOverrides = new Map(overrides.map(o => [o.account_id, o.threshold]))
    perAccountDraft.value = new Map(
      overrides.map(o => [o.account_id, { threshold: o.threshold, name: o.account_name }])
    )
    selectedAccountId.value = null
    newThreshold.value = null
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
  if (!canAddPerAccount.value) return
  const id = selectedAccountId.value as number
  const name = accountOptions.value.find(a => a.id === id)?.name ?? `#${id}`
  perAccountDraft.value.set(id, { threshold: newThreshold.value as number, name })
  // 重新赋值以触发 Map 的响应式更新
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
    if (form.value.alert_threshold == null || form.value.alert_threshold < 0) {
      appStore.showError(t('admin.accounts.balance.thresholdInvalid'))
      return
    }
    const cfg: UpstreamBalanceConfig = {
      sync_interval_seconds: form.value.sync_interval_seconds,
      alert_enabled: form.value.alert_enabled,
      alert_emails: emails,
      alert_threshold: form.value.alert_threshold
    }
    await accountsAPI.updateUpstreamBalanceConfig(cfg)

    // 专属阈值只提交有变化的：新增/修改的设置，草稿里已移除的清除
    for (const [id, v] of perAccountDraft.value) {
      if (loadedOverrides.get(id) !== v.threshold) {
        await accountsAPI.setAccountBalanceAlertThreshold(id, v.threshold)
      }
    }
    for (const id of loadedOverrides.keys()) {
      if (!perAccountDraft.value.has(id)) {
        await accountsAPI.setAccountBalanceAlertThreshold(id, null)
      }
    }
    loadedOverrides = new Map(Array.from(perAccountDraft.value, ([id, v]) => [id, v.threshold]))

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
