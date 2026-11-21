<template>
  <div
    class="cs-wrap"
    :class="{ 'cs-disabled': disabled, 'cs-error': isDraining }"
    :style="wrapStyle"
    @click="focusInput"
  >
    <!-- 隐藏 input：唯一真值来源，承载焦点/键盘/粘贴/SMS autofill；透明铺满容器，点击格子即聚焦 -->
    <input
      ref="hiddenInput"
      class="cs-hidden-input"
      type="text"
      :value="modelValue"
      :id="inputId"
      :data-testid="testid"
      :disabled="disabled"
      inputmode="numeric"
      autocomplete="one-time-code"
      :aria-label="ariaLabel"
      @input="handleInput"
      @keydown="handleKeydown"
      @select="syncCaret"
      @keyup="syncCaret"
      @click="syncCaret"
    />

    <div class="cs-slots" :style="slotsStyle">
      <div
        v-for="(_, i) in length"
        :key="i"
        class="cs-slot"
        :class="{
          'cs-slot-active': i === activeIndex && status === 'idle' && !disabled,
          'cs-slot-filled': !!digitAt(i),
          'cs-slot-success': status === 'success',
          'cs-slot-error': isDraining,
        }"
        :style="slotStyle(i)"
      >
        <span
          v-if="digitAt(i)"
          :key="`${i}-${digitAt(i)}`"
          class="cs-digit"
          :class="{ 'cs-digit-drain': isDraining }"
        >{{ digitAt(i) }}</span>
      </div>

      <!-- 自定义光标：跟随 active 格，输入框原生 caret 已隐藏 -->
      <div v-if="showCaret" class="cs-caret" :style="caretStyle"></div>

      <!-- 成功徽章：整体居中打勾，弹起动画 -->
      <transition name="cs-badge">
        <div v-if="status === 'success'" class="cs-success-badge" :style="badgeStyle">
          <svg class="cs-check" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
          </svg>
        </div>
      </transition>
    </div>

    <span class="sr-only" aria-live="polite">{{ progressText }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

/**
 * CodeSlotsInput — 分格式一次性验证码输入（ReactBits CodeSlots 的 Vue 移植）
 *
 * 架构：单个透明 input 铺满容器承载真实焦点/键盘/粘贴/autofill，视觉上渲染分格。
 * 动效全部 CSS 实现（弹性过冲曲线近似 spring），无第三方动画依赖，
 * prefers-reduced-motion 下自动降级。
 */
const props = withDefaults(
  defineProps<{
    modelValue?: string
    length?: number
    status?: 'idle' | 'error' | 'success'
    disabled?: boolean
    autofocus?: boolean
    slotSize?: number
    gap?: number
    radius?: number
    /** 隐藏 input 的 id（兼容依赖 #code 等选择器的测试与表单） */
    inputId?: string
    /** 隐藏 input 的 data-testid */
    testid?: string
    ariaLabel?: string
  }>(),
  {
    modelValue: '',
    length: 6,
    status: 'idle',
    disabled: false,
    autofocus: false,
    slotSize: 44,
    gap: 8,
    radius: 12,
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
  /** 输满时触发一次（排空清空后再次填满会重新触发） */
  complete: [code: string]
  /** 错误级联排空结束（modelValue 已清空） */
  drained: []
}>()

const { t } = useI18n()
const hiddenInput = ref<HTMLInputElement | null>(null)
const activeIndex = ref(0)
const isDraining = ref(false)
const completedOnce = ref(false)

const digitAt = (i: number) => props.modelValue[i] || ''

// success 洗入动画也做级联
const statusIsSuccess = computed(() => props.status === 'success')

// 错误级联排空：每格按 index 延迟下沉，结束后清空并回调
watch(
  () => props.status,
  (s) => {
    if (s === 'error' && props.modelValue.length > 0 && !isDraining.value) {
      isDraining.value = true
      const total = props.modelValue.length * 20 + 320
      window.setTimeout(() => {
        isDraining.value = false
        setValue('')
        emit('drained')
      }, total)
    }
    if (s === 'idle') completedOnce.value = false
  }
)

watch(
  () => props.modelValue,
  (v) => {
    if (v.length === props.length) {
      if (!completedOnce.value) {
        completedOnce.value = true
        emit('complete', v)
      }
    } else if (v.length < props.length) {
      completedOnce.value = false
    }
    // 输满后 caret 停在最后一格
    activeIndex.value = Math.min(v.length, props.length - 1)
  }
)

function setValue(v: string) {
  emit('update:modelValue', v)
}

function handleInput(e: Event) {
  const el = e.target as HTMLInputElement
  const digits = el.value.replace(/\D/g, '').slice(0, props.length)
  if (digits !== props.modelValue) setValue(digits)
  // 过滤后同步 DOM value（否则粘贴含非数字文本时 input 会残留非法字符）
  if (el.value !== digits) el.value = digits
  syncCaret()
}

function handleKeydown(e: KeyboardEvent) {
  // 左右方向键移动 active 格；其余交给原生 input 行为（Backspace 自然删末位）
  if (e.key === 'ArrowLeft' || e.key === 'ArrowRight') {
    window.setTimeout(syncCaret, 0)
  }
}

function syncCaret() {
  const el = hiddenInput.value
  if (!el) return
  const pos = el.selectionStart ?? el.value.length
  activeIndex.value = Math.min(pos, props.length - 1)
}

function focusInput() {
  hiddenInput.value?.focus()
}

onMounted(() => {
  if (props.autofocus && !props.disabled) hiddenInput.value?.focus()
})

const wrapStyle = computed(() => ({
  '--cs-size': `${props.slotSize}px`,
  '--cs-gap': `${props.gap}px`,
  '--cs-radius': `${props.radius}px`,
}))

const slotsStyle = computed(() => ({
  gap: `${props.gap}px`,
}))

const slotStyle = (i: number) => ({
  width: `${props.slotSize}px`,
  height: `${props.slotSize}px`,
  borderRadius: `${props.radius}px`,
  transitionDelay: isDraining.value ? `${i * 20}ms` : undefined,
  animationDelay: statusIsSuccess.value ? `${i * 20}ms` : undefined,
})

const showCaret = computed(
  () => props.status === 'idle' && !isDraining.value && !props.disabled && props.modelValue.length < props.length
)

const caretStyle = computed(() => ({
  left: `${activeIndex.value * (props.slotSize + props.gap) + props.slotSize / 2}px`,
  height: `${Math.round(props.slotSize * 0.45)}px`,
}))

const badgeStyle = computed(() => ({
  width: `${props.slotSize * 1.6}px`,
  height: `${props.slotSize * 1.6}px`,
}))

const progressText = computed(() => t('auth.codeslots.progress', { count: props.modelValue.length, total: props.length }))
const ariaLabel = computed(() => props.ariaLabel || t('auth.codeslots.ariaLabel'))

defineExpose({
  focus: focusInput,
  clear: () => setValue(''),
})
</script>

<style scoped>
.cs-wrap {
  position: relative;
  display: inline-block;
}

.cs-hidden-input {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  opacity: 0;
  border: none;
  outline: none;
  background: transparent;
  caret-color: transparent;
  /* 16px 起防 iOS 聚焦缩放 */
  font-size: 16px;
  color: transparent;
  cursor: pointer;
  z-index: 2;
}

.cs-slots {
  position: relative;
  display: flex;
  justify-content: center;
}

.cs-slot {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: rgb(var(--color-gray-100));
  border: 1.5px solid rgb(var(--color-gray-300));
  transition:
    border-color 0.2s ease,
    background 0.2s ease,
    transform 0.2s ease;
}

/* active 格全主题统一：半透明主色底（不用 primary-50 浅色底，避免深色下白格白字） */
.cs-slot-active {
  border-color: rgb(var(--color-primary-500));
  box-shadow: 0 0 0 3px rgb(var(--color-primary-500) / 0.18);
  background: rgb(var(--color-primary-500) / 0.1);
}

.cs-digit {
  position: relative;
  z-index: 1;
  font-size: 1.25rem;
  font-weight: 700;
  font-family: var(--font-mono, ui-monospace, monospace);
  color: rgb(var(--color-gray-900));
  animation: cs-pop 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.cs-digit {
  position: relative;
  z-index: 1;
  font-size: 1.25rem;
  font-weight: 700;
  font-family: var(--font-mono, ui-monospace, monospace);
  color: rgb(var(--color-gray-900));
  animation: cs-pop 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}

:global(.dark) .cs-digit {
  color: rgb(var(--color-dark-50, var(--color-dark-100)));
}

/* 成功时数字转白 */
.cs-slot-success .cs-digit {
  color: #fff;
}

/* 错误级联下沉排空（级联延迟由内联 transitionDelay 提供） */
.cs-digit-drain {
  animation: cs-drain 0.3s ease-in forwards;
}

.cs-caret {
  position: absolute;
  top: 50%;
  width: 2px;
  border-radius: 1px;
  transform: translateY(-50%);
  background: rgb(var(--color-primary-500));
  transition: left 0.22s cubic-bezier(0.34, 1.56, 0.64, 1);
  animation: cs-blink 1.1s step-end infinite;
  pointer-events: none;
  z-index: 1;
}

.cs-success-badge {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  background: rgb(var(--color-emerald-500));
  color: #fff;
  box-shadow: 0 8px 24px rgb(var(--color-emerald-500) / 0.4);
  z-index: 3;
}

.cs-check {
  width: 55%;
  height: 55%;
}

.cs-badge-enter-active {
  animation: cs-badge-pop 0.35s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.cs-badge-leave-active {
  transition: opacity 0.15s ease;
}

.cs-badge-enter-from,
.cs-badge-leave-to {
  opacity: 0;
}

.cs-disabled .cs-slot {
  opacity: 0.55;
  cursor: not-allowed;
}

.cs-disabled .cs-hidden-input {
  cursor: not-allowed;
}

@keyframes cs-pop {
  0% {
    transform: translateY(8px) scale(0.6);
    opacity: 0;
  }
  100% {
    transform: translateY(0) scale(1);
    opacity: 1;
  }
}

@keyframes cs-drain {
  0% {
    transform: translateY(0);
    opacity: 1;
  }
  100% {
    transform: translateY(10px);
    opacity: 0;
  }
}

@keyframes cs-wash {
  0% {
    clip-path: inset(50% 0 50% 0);
  }
  100% {
    clip-path: inset(0 0 0 0);
  }
}

@keyframes cs-blink {
  0%,
  49.9% {
    opacity: 1;
  }
  50%,
  100% {
    opacity: 0;
  }
}

@keyframes cs-badge-pop {
  0% {
    transform: translate(-50%, -50%) scale(0.3);
    opacity: 0;
  }
  100% {
    transform: translate(-50%, -50%) scale(1);
    opacity: 1;
  }
}

@media (prefers-reduced-motion: reduce) {
  .cs-digit,
  .cs-digit-drain,
  .cs-slot-success::before,
  .cs-caret,
  .cs-badge-enter-active {
    animation: none !important;
    transition: none !important;
  }

  .cs-slot-success::before {
    clip-path: inset(0 0 0 0);
  }
}
</style>

<!-- 深色覆盖：非 scoped（Vue scoped 编译器会静默丢弃 `:global(.dark) .local` 混合写法）；
     cs- 前缀为组件专属类名，全局作用无冲突风险 -->
<style>
.dark .cs-slot {
  background: rgb(var(--color-dark-800));
  border-color: rgb(var(--color-dark-600));
}

.dark .cs-slot-active {
  border-color: rgb(var(--color-primary-400));
  background: rgb(var(--color-primary-500) / 0.16);
}

.dark .cs-digit {
  color: #fff;
}
</style>
