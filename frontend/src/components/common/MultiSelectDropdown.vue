<template>
  <div class="relative" ref="containerRef">
    <button
      type="button"
      class="select-trigger"
      :class="isOpen && 'select-trigger-open'"
      :aria-expanded="isOpen"
      aria-haspopup="true"
      :data-testid="testId"
      @click="toggle"
    >
      <span class="select-value truncate">
        {{ displayLabel }}
      </span>
      <span v-if="modelValue.length > 0" class="mr-1 rounded bg-primary-100 px-1.5 text-xs font-medium text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">
        {{ modelValue.length }}
      </span>
      <span class="select-icon">
        <Icon name="chevronDown" size="md" :class="['transition-transform duration-200', isOpen && 'rotate-180']" />
      </span>
    </button>

    <Teleport to="body">
      <Transition name="select-dropdown">
        <div
          v-if="isOpen"
          class="select-dropdown-portal"
          :style="dropdownStyle"
          role="listbox"
          aria-multiselectable="true"
          @click.stop
          @mousedown.stop
        >
          <div v-if="searchable" class="select-search">
            <Icon name="search" size="sm" class="text-gray-400" />
            <input
              ref="searchInputRef"
              v-model="searchQuery"
              type="text"
              class="select-search-input"
              :placeholder="t('common.searchPlaceholder')"
            />
          </div>
          <div class="select-options max-h-64">
            <div
              v-for="option in filteredOptions"
              :key="String(option.value)"
              role="option"
              :aria-selected="isSelected(option.value)"
              class="select-option"
              :class="isSelected(option.value) && 'select-option-selected'"
              @click.stop="toggleOption(option.value)"
            >
              <input
                type="checkbox"
                :checked="isSelected(option.value)"
                class="h-3.5 w-3.5 shrink-0 rounded border-gray-300 text-primary-500 focus:ring-primary-500 dark:border-dark-500"
                @click.stop.prevent="toggleOption(option.value)"
              />
              <span class="select-option-label">{{ option.label }}</span>
              <Icon v-if="isSelected(option.value)" name="check" size="sm" class="text-primary-500" :stroke-width="2" />
            </div>
            <div v-if="filteredOptions.length === 0" class="select-empty">
              {{ t('common.noData') }}
            </div>
          </div>
          <div class="flex items-center justify-between border-t border-gray-100 px-3 py-2 dark:border-dark-600">
            <button type="button" class="text-xs text-gray-500 hover:text-gray-700 dark:text-gray-400" @click.stop="clearAll">
              {{ t('common.clear') }}
            </button>
            <span class="text-xs text-gray-400">{{ t('common.selectedCount', { count: modelValue.length }) }}</span>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

interface Option {
  value: string | number
  label: string
}

const props = withDefaults(defineProps<{
  modelValue: (string | number)[]
  options: Option[]
  placeholder?: string
  searchable?: boolean
  testId?: string
}>(), {
  placeholder: '',
  searchable: true,
  testId: undefined
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: (string | number)[]): void
}>()

const { t } = useI18n()

const containerRef = ref<HTMLElement | null>(null)
const searchInputRef = ref<HTMLInputElement | null>(null)
const isOpen = ref(false)
const searchQuery = ref('')
const dropdownStyle = ref<Record<string, string>>({})

const displayLabel = computed(() => {
  if (props.modelValue.length === 0) {
    return props.placeholder || t('common.all')
  }
  const labels = props.modelValue
    .map((v) => props.options.find((o) => o.value === v)?.label)
    .filter(Boolean) as string[]
  if (labels.length <= 2) return labels.join(', ')
  return `${labels[0]}, ${labels[1]} …`
})

const filteredOptions = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return props.options
  return props.options.filter((o) => o.label.toLowerCase().includes(q))
})

const isSelected = (value: string | number) => props.modelValue.includes(value)

const toggleOption = (value: string | number) => {
  const next = isSelected(value)
    ? props.modelValue.filter((v) => v !== value)
    : [...props.modelValue, value]
  emit('update:modelValue', next)
}

const clearAll = () => emit('update:modelValue', [])

const toggle = () => {
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    searchQuery.value = ''
    // 下拉挂到触发按钮正下方（Teleport 到 body 后需要手动定位）
    const rect = containerRef.value?.getBoundingClientRect()
    if (rect) {
      const width = Math.max(rect.width, 240)
      const left = Math.min(rect.left, window.innerWidth - width - 8)
      dropdownStyle.value = {
        position: 'fixed',
        top: `${rect.bottom + 4}px`,
        left: `${Math.max(8, left)}px`,
        width: `${width}px`,
        zIndex: '60'
      }
    }
    requestAnimationFrame(() => searchInputRef.value?.focus())
  }
}

const onDocumentClick = (event: MouseEvent) => {
  if (!isOpen.value) return
  const target = event.target as Node
  if (containerRef.value?.contains(target)) return
  const portal = document.querySelector('.select-dropdown-portal')
  if (portal?.contains(target)) return
  isOpen.value = false
}

onMounted(() => document.addEventListener('mousedown', onDocumentClick))
onBeforeUnmount(() => document.removeEventListener('mousedown', onDocumentClick))
</script>
