<template>
  <BaseDialog :show="show" :title="detail?.name || t('admin.upstreams.detail.title')" width="extra-wide" @close="close">
    <div v-if="loading" class="flex items-center justify-center py-10">
      <LoadingSpinner />
    </div>

    <div v-else-if="detail" class="space-y-4">
      <!-- 概览 -->
      <div class="flex flex-wrap items-center gap-x-6 gap-y-2 rounded-lg bg-gray-50 px-4 py-3 text-sm dark:bg-dark-700/50">
        <span v-if="detail.base_url" class="text-gray-600 dark:text-gray-300">
          <span class="text-gray-400">{{ t('admin.upstreams.detail.baseUrl') }}:</span> {{ detail.base_url }}
        </span>
        <span v-if="detail.website" class="text-gray-600 dark:text-gray-300">
          <span class="text-gray-400">{{ t('admin.upstreams.detail.website') }}:</span>
          <a :href="detail.website" target="_blank" rel="noopener noreferrer" class="text-primary-600 hover:underline">{{ detail.website }}</a>
        </span>
        <span v-if="detail.contact" class="text-gray-600 dark:text-gray-300">
          <span class="text-gray-400">{{ t('admin.upstreams.detail.contact') }}:</span> {{ detail.contact }}
        </span>
        <span v-if="detail.note" class="text-gray-600 dark:text-gray-300">
          <span class="text-gray-400">{{ t('admin.upstreams.detail.note') }}:</span> {{ detail.note }}
        </span>
        <button class="btn btn-secondary ml-auto px-3 py-1.5 text-sm" @click="openEdit">
          {{ t('admin.upstreams.actions.edit') }}
        </button>
      </div>

      <!-- Tabs -->
      <div class="flex gap-2 border-b border-gray-100 dark:border-dark-600">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="border-b-2 px-3 py-2 text-sm font-medium transition-colors"
          :class="activeTab === tab.key
            ? 'border-primary-500 text-primary-600 dark:text-primary-400'
            : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400'"
          :data-testid="`upstream-tab-${tab.key}`"
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
          <span class="ml-1 text-xs text-gray-400">{{ tab.count }}</span>
        </button>
      </div>

      <!-- 账号 -->
      <div v-if="activeTab === 'accounts'" class="overflow-x-auto">
        <p v-if="detail.accounts.length === 0" class="py-6 text-center text-sm text-gray-400">
          {{ t('admin.upstreams.detail.noAccounts') }}
        </p>
        <table v-else class="min-w-full text-sm">
          <thead class="text-left text-xs uppercase text-gray-500 dark:text-gray-400">
            <tr>
              <th class="px-3 py-2">{{ t('admin.upstreams.accounts.name') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.accounts.platform') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.accounts.status') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.accounts.rate') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.accounts.balance') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.detail.probedRate') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.accounts.groups') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.accounts.mapping') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-600">
            <tr v-for="account in detail.accounts" :key="account.id">
              <td class="px-3 py-2 font-medium text-gray-900 dark:text-white">{{ account.name }}</td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-300">{{ account.platform }}</td>
              <td class="px-3 py-2">
                <span class="rounded-full px-2 py-0.5 text-xs"
                  :class="account.status === 'active'
                    ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
                    : 'bg-gray-100 text-gray-600 dark:bg-dark-600 dark:text-gray-300'">
                  {{ account.status }}
                </span>
              </td>
              <td class="px-3 py-2 font-mono text-gray-900 dark:text-gray-100">×{{ account.rate_multiplier }}</td>
              <td class="px-3 py-2 font-mono text-gray-900 dark:text-gray-100">
                {{ formatMoney(account.upstream_balance) }}
              </td>
              <td class="px-3 py-2 font-mono text-gray-600 dark:text-gray-300" :title="t('admin.upstreams.detail.probedHint')">
                {{ account.probed_group_rate_multiplier != null ? `×${account.probed_group_rate_multiplier}` : '—' }}
                <span v-if="account.probed_peak_multiplier" class="text-xs text-gray-400">
                  / ★×{{ account.probed_peak_multiplier }}
                </span>
              </td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-300">
                {{ account.group_names.join(', ') || '—' }}
              </td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-300">
                {{ t('admin.upstreams.detail.mappedModels', { count: account.mapped_model_count }) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 上游分组 -->
      <div v-if="activeTab === 'groups'" class="space-y-3">
        <div class="flex justify-end">
          <button class="btn btn-secondary px-3 py-1.5 text-sm" data-testid="upstream-add-group" @click="openGroupForm()">
            {{ t('admin.upstreams.groups.add') }}
          </button>
        </div>
        <p v-if="detail.groups.length === 0" class="py-6 text-center text-sm text-gray-400">
          {{ t('admin.upstreams.detail.noGroups') }}
        </p>
        <table v-else class="min-w-full text-sm">
          <thead class="text-left text-xs uppercase text-gray-500 dark:text-gray-400">
            <tr>
              <th class="px-3 py-2">{{ t('admin.upstreams.groups.name') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.groups.rate') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.groups.peak') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.groups.peakWindow') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.groups.source') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.groups.note') }}</th>
              <th class="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-600">
            <tr v-for="group in detail.groups" :key="group.id">
              <td class="px-3 py-2 font-medium text-gray-900 dark:text-white">{{ group.name }}</td>
              <td class="px-3 py-2 font-mono">{{ group.rate_multiplier != null ? `×${group.rate_multiplier}` : '—' }}</td>
              <td class="px-3 py-2 font-mono">{{ group.peak_rate_multiplier != null ? `×${group.peak_rate_multiplier}` : '—' }}</td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-300">
                {{ group.peak_start || group.peak_end ? `${group.peak_start || '?'} - ${group.peak_end || '?'}` : '—' }}
              </td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-300">{{ t(`admin.upstreams.groups.sourceValues.${group.source}`) }}</td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-300">{{ group.note || '—' }}</td>
              <td class="px-3 py-2 text-right">
                <button class="text-xs text-primary-600 hover:underline" @click="openGroupForm(group)">
                  {{ t('admin.upstreams.groups.edit') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 模型价格台账 -->
      <div v-if="activeTab === 'prices'" class="space-y-3">
        <div class="flex justify-end gap-2">
          <button class="btn btn-secondary px-3 py-1.5 text-sm" data-testid="upstream-import-prices" @click="openImport">
            {{ t('admin.upstreams.prices.import') }}
          </button>
          <button class="btn btn-secondary px-3 py-1.5 text-sm" data-testid="upstream-add-price" @click="openPriceForm()">
            {{ t('admin.upstreams.prices.add') }}
          </button>
        </div>
        <p v-if="detail.model_prices.length === 0" class="py-6 text-center text-sm text-gray-400">
          {{ t('admin.upstreams.detail.noPrices') }}
        </p>
        <table v-else class="min-w-full text-sm">
          <thead class="text-left text-xs uppercase text-gray-500 dark:text-gray-400">
            <tr>
              <th class="px-3 py-2">{{ t('admin.upstreams.prices.upstreamModel') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.prices.modelName') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.prices.group') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.prices.input') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.prices.output') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.prices.rate') }}</th>
              <th class="px-3 py-2">{{ t('admin.upstreams.prices.source') }}</th>
              <th class="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-600">
            <tr v-for="price in detail.model_prices" :key="price.id">
              <td class="px-3 py-2 font-mono text-xs text-gray-900 dark:text-white">{{ price.upstream_model }}</td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-300">{{ price.model_name || '—' }}</td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-300">{{ groupName(price.upstream_group_id) }}</td>
              <td class="px-3 py-2 font-mono">{{ formatPerMillion(price.input_price_per_token) }}</td>
              <td class="px-3 py-2 font-mono">{{ formatPerMillion(price.output_price_per_token) }}</td>
              <td class="px-3 py-2 font-mono">{{ price.rate_multiplier != null ? `×${price.rate_multiplier}` : '—' }}</td>
              <td class="px-3 py-2 text-gray-600 dark:text-gray-300">{{ t(`admin.upstreams.prices.sourceValues.${price.source}`) }}</td>
              <td class="px-3 py-2 text-right">
                <button class="mr-2 text-xs text-primary-600 hover:underline" @click="openPriceForm(price)">
                  {{ t('admin.upstreams.prices.edit') }}
                </button>
                <button class="text-xs text-red-500 hover:underline" @click="removePrice(price.id)">
                  {{ t('common.delete') }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button type="button" class="btn btn-secondary" @click="close">{{ t('common.close') }}</button>
      </div>
    </template>
  </BaseDialog>

  <!-- 上游分组编辑 -->
  <BaseDialog :show="groupFormVisible" :title="t('admin.upstreams.groups.add')" @close="groupFormVisible = false">
    <div class="space-y-4">
      <div>
        <label class="input-label">{{ t('admin.upstreams.groups.name') }}</label>
        <input v-model="groupForm.name" class="input mt-1 w-full" data-testid="upstream-group-name" />
      </div>
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.upstreams.groups.rate') }}</label>
          <input v-model.number="groupForm.rate_multiplier" type="number" step="0.0001" min="0" class="input mt-1 w-full" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.upstreams.groups.peak') }}</label>
          <input v-model.number="groupForm.peak_rate_multiplier" type="number" step="0.0001" min="0" class="input mt-1 w-full" />
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('admin.upstreams.groups.note') }}</label>
        <input v-model="groupForm.note" class="input mt-1 w-full" />
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" @click="groupFormVisible = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="saving || !groupForm.name.trim()" @click="saveGroup">
          {{ t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <!-- 台账编辑 -->
  <BaseDialog :show="priceFormVisible" :title="t('admin.upstreams.prices.add')" @close="priceFormVisible = false">
    <div class="space-y-4">
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.upstreams.prices.upstreamModel') }}</label>
          <input v-model="priceForm.upstream_model" class="input mt-1 w-full" data-testid="upstream-price-model" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.upstreams.prices.modelName') }}</label>
          <input v-model="priceForm.model_name" class="input mt-1 w-full" />
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('admin.upstreams.prices.group') }}</label>
        <Select
          v-model="priceForm.upstream_group_id"
          :options="groupOptions"
          :placeholder="t('admin.upstreams.prices.group')"
          class="mt-1 w-full"
        />
      </div>
      <div class="grid gap-4 sm:grid-cols-3">
        <div>
          <label class="input-label">{{ t('admin.upstreams.prices.input') }}</label>
          <input v-model.number="priceForm.input_per_million" type="number" step="0.000001" min="0" class="input mt-1 w-full" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.upstreams.prices.output') }}</label>
          <input v-model.number="priceForm.output_per_million" type="number" step="0.000001" min="0" class="input mt-1 w-full" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.upstreams.prices.rate') }}</label>
          <input v-model.number="priceForm.rate_multiplier" type="number" step="0.0001" min="0" class="input mt-1 w-full" />
        </div>
      </div>
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.upstreams.prices.perMillionHint') }}</p>
    </div>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" @click="priceFormVisible = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="saving || !priceForm.upstream_model.trim()" @click="savePrice">
          {{ t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <!-- 批量导入 -->
  <BaseDialog :show="importVisible" :title="t('admin.upstreams.prices.import')" @close="importVisible = false">
    <div class="space-y-3">
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.upstreams.prices.importHint') }}</p>
      <textarea
        v-model="importText"
        rows="10"
        class="input w-full font-mono text-xs"
        :placeholder="t('admin.upstreams.prices.importPlaceholder')"
        data-testid="upstream-price-import-text"
      ></textarea>
    </div>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" @click="importVisible = false">{{ t('common.cancel') }}</button>
        <button class="btn btn-primary" :disabled="saving || !importText.trim()" @click="runImport">
          {{ t('common.confirm') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <UpstreamFormDialog v-model:show="editFormVisible" :upstream="detail" @saved="reload" />
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Select from '@/components/common/Select.vue'
import UpstreamFormDialog from './UpstreamFormDialog.vue'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type {
  UpstreamDetail,
  UpstreamGroup,
  UpstreamModelPrice,
  UpstreamModelPriceInput
} from '@/api/admin/upstreams'

const props = defineProps<{
  show: boolean
  upstreamId: number | null
}>()

const emit = defineEmits<{
  (e: 'update:show', value: boolean): void
  (e: 'changed'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const detail = ref<UpstreamDetail | null>(null)
const activeTab = ref<'accounts' | 'groups' | 'prices'>('accounts')

const tabs = computed(() => [
  { key: 'accounts' as const, label: t('admin.upstreams.detail.tabs.accounts'), count: detail.value?.accounts.length ?? 0 },
  { key: 'groups' as const, label: t('admin.upstreams.detail.tabs.groups'), count: detail.value?.groups.length ?? 0 },
  { key: 'prices' as const, label: t('admin.upstreams.detail.tabs.prices'), count: detail.value?.model_prices.length ?? 0 }
])

const groupOptions = computed(() => [
  { value: null, label: '—' },
  ...(detail.value?.groups ?? []).map((g) => ({ value: g.id, label: g.name }))
])

const groupName = (id: number | null): string => {
  if (id == null) return '—'
  return detail.value?.groups.find((g) => g.id === id)?.name ?? `#${id}`
}

const formatMoney = (value: number | null): string =>
  value == null ? '—' : `$${value.toFixed(2)}`

const formatPerMillion = (perToken: number | null): string =>
  perToken == null ? '—' : `$${(perToken * 1e6).toFixed(4)}`

const load = async () => {
  if (!props.upstreamId) return
  loading.value = true
  try {
    detail.value = await adminAPI.upstreams.getById(props.upstreamId)
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.messages.loadFailed'))
  } finally {
    loading.value = false
  }
}

const reload = async () => {
  await load()
  emit('changed')
}

watch(
  () => [props.show, props.upstreamId] as const,
  ([show]) => {
    if (show) {
      activeTab.value = 'accounts'
      void load()
    }
  },
  { immediate: true }
)

const close = () => emit('update:show', false)

// ---------- 编辑上游 ----------
const editFormVisible = ref(false)
const openEdit = () => {
  editFormVisible.value = true
}

// ---------- 上游分组 ----------
const groupFormVisible = ref(false)
const groupForm = reactive({
  id: 0,
  name: '',
  rate_multiplier: null as number | null,
  peak_rate_multiplier: null as number | null,
  note: ''
})

const openGroupForm = (group?: UpstreamGroup) => {
  groupForm.id = group?.id ?? 0
  groupForm.name = group?.name ?? ''
  groupForm.rate_multiplier = group?.rate_multiplier ?? null
  groupForm.peak_rate_multiplier = group?.peak_rate_multiplier ?? null
  groupForm.note = group?.note ?? ''
  groupFormVisible.value = true
}

const saveGroup = async () => {
  if (!props.upstreamId || saving.value || !groupForm.name.trim()) return
  saving.value = true
  try {
    await adminAPI.upstreams.upsertGroup(props.upstreamId, {
      id: groupForm.id || undefined,
      name: groupForm.name.trim(),
      rate_multiplier: groupForm.rate_multiplier,
      peak_rate_multiplier: groupForm.peak_rate_multiplier,
      note: groupForm.note.trim()
    })
    appStore.showSuccess(t('admin.upstreams.messages.groupSaved'))
    groupFormVisible.value = false
    await reload()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.messages.saveFailed'))
  } finally {
    saving.value = false
  }
}

// ---------- 台账 ----------
const priceFormVisible = ref(false)
const priceForm = reactive({
  id: 0,
  upstream_model: '',
  model_name: '',
  upstream_group_id: null as number | null,
  input_per_million: null as number | null,
  output_per_million: null as number | null,
  rate_multiplier: null as number | null
})

const openPriceForm = (price?: UpstreamModelPrice) => {
  priceForm.id = price?.id ?? 0
  priceForm.upstream_model = price?.upstream_model ?? ''
  priceForm.model_name = price?.model_name ?? ''
  priceForm.upstream_group_id = price?.upstream_group_id ?? null
  priceForm.input_per_million =
    price?.input_price_per_token != null ? price.input_price_per_token * 1e6 : null
  priceForm.output_per_million =
    price?.output_price_per_token != null ? price.output_price_per_token * 1e6 : null
  priceForm.rate_multiplier = price?.rate_multiplier ?? null
  priceFormVisible.value = true
}

const toPerToken = (perMillion: number | null): number | null =>
  perMillion == null ? null : perMillion / 1e6

const savePrice = async () => {
  if (!props.upstreamId || saving.value || !priceForm.upstream_model.trim()) return
  saving.value = true
  try {
    await adminAPI.upstreams.upsertModelPrice(props.upstreamId, {
      id: priceForm.id || undefined,
      upstream_model: priceForm.upstream_model.trim(),
      model_name: priceForm.model_name.trim(),
      upstream_group_id: priceForm.upstream_group_id,
      input_price_per_token: toPerToken(priceForm.input_per_million),
      output_price_per_token: toPerToken(priceForm.output_per_million),
      rate_multiplier: priceForm.rate_multiplier
    })
    appStore.showSuccess(t('admin.upstreams.messages.priceSaved'))
    priceFormVisible.value = false
    await reload()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.messages.saveFailed'))
  } finally {
    saving.value = false
  }
}

const removePrice = async (priceId: number) => {
  if (!props.upstreamId) return
  try {
    await adminAPI.upstreams.deleteModelPrice(props.upstreamId, priceId)
    await reload()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.messages.saveFailed'))
  }
}

// ---------- 批量导入 ----------
const importVisible = ref(false)
const importText = ref('')

const openImport = () => {
  importText.value = ''
  importVisible.value = true
}

const runImport = async () => {
  if (!props.upstreamId || saving.value) return
  let items: UpstreamModelPriceInput[]
  try {
    const parsed = JSON.parse(importText.value)
    if (!Array.isArray(parsed)) throw new Error('not an array')
    items = parsed
  } catch {
    appStore.showError(t('admin.upstreams.messages.saveFailed'))
    return
  }
  saving.value = true
  try {
    const result = await adminAPI.upstreams.importModelPrices(props.upstreamId, items, 'import')
    appStore.showSuccess(t('admin.upstreams.messages.imported', { count: result.upserted }))
    if (result.unmatched.length > 0) {
      appStore.showError(
        t('admin.upstreams.prices.unmatchedHint', {
          count: result.unmatched.length,
          models: result.unmatched.slice(0, 5).join(', ')
        })
      )
    }
    importVisible.value = false
    await reload()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.upstreams.messages.saveFailed'))
  } finally {
    saving.value = false
  }
}
</script>
