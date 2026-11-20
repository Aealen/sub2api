/**
 * 主题感知的图表色板
 *
 * 系列色从 CSS 变量实时读取（style.css 的 primary 板与 brand 常量），因此：
 * - teal 经典主题：青粉系（primary=teal）
 * - aowu 主题：紫粉青系（primary=violet），与 LOGO 品牌三色对齐
 * 切换主题时由组件的 computed 依赖 appStore.brandTheme 触发重算，
 * vue-chartjs 检测到 data 变化自动重绘，无需手动 update。
 */
import { computed } from 'vue'
import { useAppStore } from '@/stores/app'

/** 读取根元素上的 RGB 通道变量，返回可直接用于 canvas 的颜色字符串 */
function cssRgb(varName: string): string {
  const raw = getComputedStyle(document.documentElement).getPropertyValue(varName).trim()
  return raw ? `rgb(${raw})` : '#94a3b8'
}

/**
 * 12 色系列板：主色亮/深与品牌粉、品牌青交替，相邻色相与明度错开保证区分度。
 * 次序刻意做「主色亮 → 粉 → 青 → 主色浅 → 浅粉 → 主色 → 浅青 → 主色深 → 深青 → 最浅主色」的明度跳跃。
 */
function buildSeriesPalette(): string[] {
  return [
    cssRgb('--color-primary-400'),
    cssRgb('--color-brand-pink'),
    cssRgb('--color-brand-teal-400'),
    cssRgb('--color-primary-300'),
    cssRgb('--color-brand-pink-light'),
    cssRgb('--color-primary-500'),
    cssRgb('--color-brand-teal-200'),
    cssRgb('--color-primary-600'),
    cssRgb('--color-brand-teal-600'),
    cssRgb('--color-primary-200')
  ]
}

/**
 * 图表系列色（响应式）：依赖 brandTheme，主题切换时返回新色板。
 * @param count 需要的系列数（取色板前 N 个，超出部分循环复用明度变体）
 */
export function useChartSeries(count: number) {
  const appStore = useAppStore()
  return computed(() => {
    void appStore.brandTheme // 建立响应依赖：主题切换触发重算
    const base = buildSeriesPalette()
    if (count <= base.length) return base.slice(0, count)
    // 超出色板容量时循环复用（同色重复在图例中可接受，优于硬编码杂色）
    const out: string[] = []
    for (let i = 0; i < count; i++) out.push(base[i % base.length])
    return out
  })
}

/**
 * 用量趋势图的语义系列色（输入/输出/缓存写入/缓存读取/命中率）。
 * 语义映射：输入=主色、输出=品牌青、缓存写入=品牌粉、缓存读取=主色浅、命中率=主色深。
 */
export function useTrendSeriesColors() {
  const appStore = useAppStore()
  return computed(() => {
    void appStore.brandTheme
    return {
      input: cssRgb('--color-primary-500'),
      output: cssRgb('--color-brand-teal-400'),
      cacheCreation: cssRgb('--color-brand-pink'),
      cacheRead: cssRgb('--color-primary-300'),
      cacheHitRate: cssRgb('--color-primary-600')
    }
  })
}
