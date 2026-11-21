<template>
  <div class="auth-shell relative flex min-h-screen items-center justify-center overflow-hidden p-4">
    <!-- Background -->
    <div
      class="absolute inset-0 bg-gradient-to-br from-gray-50 via-primary-50/30 to-gray-100 dark:from-dark-950 dark:via-dark-900 dark:to-dark-950"
    ></div>

    <!-- Decorative Elements -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <!-- Gradient Orbs (slow drift) -->
      <div class="auth-orb auth-orb-a absolute -right-40 -top-40 h-80 w-80 rounded-full blur-3xl"></div>
      <div class="auth-orb auth-orb-b absolute -bottom-40 -left-40 h-80 w-80 rounded-full blur-3xl"></div>
      <div class="auth-orb auth-orb-c absolute left-1/2 top-1/2 h-96 w-96 -translate-x-1/2 -translate-y-1/2 rounded-full blur-3xl"></div>

      <!-- Grid Pattern -->
      <div class="auth-grid absolute inset-0 bg-[size:64px_64px]"></div>
    </div>

    <!-- Content Container -->
    <div class="relative z-10 w-full max-w-md">
      <!-- Logo/Brand -->
      <div class="mb-8 text-center">
        <!-- Custom Logo or Default Logo -->
        <template v-if="settingsLoaded">
          <div class="auth-logo mb-4 inline-flex h-16 w-16 items-center justify-center overflow-hidden rounded-2xl shadow-lg shadow-primary-500/30">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <h1 class="text-gradient mb-2 text-3xl font-bold">
            {{ siteName }}
          </h1>
          <p class="text-sm text-gray-500 dark:text-dark-400">
            {{ siteSubtitle }}
          </p>
        </template>
      </div>

      <!-- Card Container -->
      <div class="auth-card card-glass rounded-2xl p-8 shadow-glass">
        <slot />
      </div>

      <!-- Footer Links -->
      <div class="mt-6 text-center text-sm">
        <slot name="footer" />
      </div>

      <!-- Copyright -->
      <div class="mt-8 text-center text-xs text-gray-400 dark:text-dark-500">
        &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.text-gradient {
  @apply bg-gradient-to-r from-primary-600 to-primary-500 bg-clip-text text-transparent;
}
</style>

<!-- 非 scoped：通过 .auth-shell 作用域升级三个 auth 页共用的表单视觉语言（不影响其他页面） -->
<style>
.auth-shell {
  /* 主题感知的氛围色（aowu 主题下换紫粉光球） */
  --auth-orb-a: rgb(var(--color-primary-400) / 0.22);
  --auth-orb-b: rgb(var(--color-primary-500) / 0.16);
  --auth-orb-c: rgb(var(--color-primary-300) / 0.12);
  --auth-grid: rgb(var(--color-primary-500) / 0.035);
}

[data-theme='aowu'] .auth-shell {
  --auth-orb-a: rgb(var(--color-primary-400) / 0.26);
  --auth-orb-b: rgb(236 72 153 / 0.14);
  --auth-orb-c: rgb(167 139 250 / 0.14);
  --auth-grid: rgb(var(--color-primary-500) / 0.04);
}

.auth-orb-a {
  background: var(--auth-orb-a);
  animation: auth-drift 14s ease-in-out infinite;
}

.auth-orb-b {
  background: var(--auth-orb-b);
  animation: auth-drift 18s ease-in-out infinite reverse;
}

.auth-orb-c {
  background: var(--auth-orb-c);
  animation: auth-drift 22s ease-in-out infinite;
  animation-delay: -6s;
}

.auth-grid {
  background-image:
    linear-gradient(var(--auth-grid) 1px, transparent 1px),
    linear-gradient(90deg, var(--auth-grid) 1px, transparent 1px);
}

.auth-logo {
  animation: auth-float 3.6s ease-in-out infinite;
}

/* ---- 卡片：顶部渐变高光描边 + 入场 ---- */
.auth-card {
  animation: auth-slide-up 0.5s cubic-bezier(0.22, 1, 0.36, 1) both;
  position: relative;
}

.auth-card::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  padding: 1px;
  background: linear-gradient(
    140deg,
    rgb(var(--color-primary-400) / 0.55),
    rgb(var(--color-primary-500) / 0.12) 38%,
    rgb(236 72 153 / 0.22) 75%,
    rgb(var(--color-primary-400) / 0.4)
  );
  -webkit-mask:
    linear-gradient(#fff 0 0) content-box,
    linear-gradient(#fff 0 0);
  -webkit-mask-composite: xor;
  mask:
    linear-gradient(#fff 0 0) content-box,
    linear-gradient(#fff 0 0);
  mask-composite: exclude;
  pointer-events: none;
}

/* ---- 表单视觉语言（仅 auth 页生效） ---- */
.auth-shell .input {
  border-radius: 1rem;
  transition:
    border-color 0.2s ease,
    box-shadow 0.25s ease,
    background 0.2s ease;
}

.auth-shell .input:focus {
  border-color: rgb(var(--color-primary-400));
  box-shadow:
    0 0 0 4px rgb(var(--color-primary-500) / 0.12),
    0 0 24px rgb(var(--color-primary-500) / 0.14);
}

/* CTA 按钮渐变高光 + hover 扫光 */
.auth-shell .btn-primary {
  position: relative;
  overflow: hidden;
}

.auth-shell .btn-primary::after {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  width: 45%;
  background: linear-gradient(
    100deg,
    transparent,
    rgb(255 255 255 / 0.22),
    transparent
  );
  transform: translateX(-120%) skewX(-12deg);
  pointer-events: none;
}

.auth-shell .btn-primary:hover::after {
  transform: translateX(320%) skewX(-12deg);
  transition: transform 0.7s ease;
}

.auth-shell .btn-primary:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow:
    0 10px 24px rgb(var(--color-primary-500) / 0.35),
    0 4px 10px rgb(var(--color-primary-500) / 0.25);
}

/* 分格验证码与页面动效节奏一致 */
.auth-shell .cs-slot {
  border-radius: 0.85rem;
}

.auth-shell .cs-wrap {
  transition: transform 0.2s ease;
}

@keyframes auth-drift {
  0%,
  100% {
    transform: translate(0, 0) scale(1);
  }
  50% {
    transform: translate(28px, -22px) scale(1.08);
  }
}

@keyframes auth-float {
  0%,
  100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-6px);
  }
}

@keyframes auth-slide-up {
  from {
    opacity: 0;
    transform: translateY(16px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .auth-orb-a,
  .auth-orb-b,
  .auth-orb-c,
  .auth-logo,
  .auth-card {
    animation: none !important;
  }
}
</style>
