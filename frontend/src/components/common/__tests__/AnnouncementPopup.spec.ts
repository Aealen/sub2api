import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import AnnouncementPopup from '../AnnouncementPopup.vue'
import { useAnnouncementStore } from '@/stores/announcements'

const announcementMarkdownStyles = readFileSync(
  resolve(process.cwd(), 'src/styles/announcement-markdown.css'),
  'utf8',
)

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const announcement = {
  id: 1,
  title: 'Preview announcement',
  content: '## Preview heading\n\n<div>HTML content</div><script>window.__xss = true</script>',
  status: 'draft' as const,
  notify_mode: 'popup' as const,
  targeting: { any_of: [] },
  created_at: '2026-07-24T07:30:00Z',
  updated_at: '2026-07-24T07:30:00Z',
}

describe('AnnouncementPopup', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    document.body.innerHTML = ''
    document.body.style.overflow = ''
  })

  it('renders mixed Markdown and HTML inside the shared styled container', async () => {
    const store = useAnnouncementStore()
    store.currentPopup = {
      id: 1,
      title: 'Mixed content announcement',
      content: [
        '## Markdown heading',
        '',
        '<div><h3>HTML heading</h3><ul><li>HTML list item</li></ul></div>',
        '',
        '<table><thead><tr><th>Status</th></tr></thead><tbody><tr><td>OK</td></tr></tbody></table>',
        '<script>window.__announcementXss = true</script>',
      ].join('\n'),
      notify_mode: 'popup',
      created_at: '2026-07-24T07:30:00Z',
      updated_at: '2026-07-24T07:30:00Z',
    }

    const wrapper = mount(AnnouncementPopup)
    await wrapper.vm.$nextTick()

    const content = document.body.querySelector('.markdown-body')
    expect(content?.querySelector('h2')?.textContent).toBe('Markdown heading')
    expect(content?.querySelector('h3')?.textContent).toBe('HTML heading')
    expect(content?.querySelector('li')?.textContent).toBe('HTML list item')
    expect(content?.querySelector('table td')?.textContent).toBe('OK')
    expect(content?.querySelector('script')).toBeNull()

    wrapper.unmount()
  })

  it.each(['h2', 'h3', 'ul', 'li', 'blockquote', 'table', 'th', 'td', 'code'])(
    'loads a shared style rule for mixed-content <%s> elements',
    (element) => {
      expect(announcementMarkdownStyles).toContain(`.markdown-body ${element}`)
    },
  )

  it('previews an admin announcement without marking it as read', async () => {
    const store = useAnnouncementStore()
    const dismissPopup = vi.spyOn(store, 'dismissPopup')
    const wrapper = mount(AnnouncementPopup, {
      props: {
        announcement,
        preview: true,
      },
    })

    expect(document.body.textContent).toContain('Preview announcement')
    expect(document.body.querySelector('.markdown-body h2')?.textContent).toBe('Preview heading')
    expect(document.body.querySelector('.markdown-body script')).toBeNull()
    expect(document.body.textContent).toContain('common.close')

    const dismissButton = document.body.querySelector<HTMLButtonElement>(
      '[data-testid="announcement-popup-dismiss"]',
    )
    dismissButton?.click()
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('close')).toHaveLength(1)
    expect(dismissPopup).not.toHaveBeenCalled()

    await wrapper.setProps({ announcement: null })
    expect(document.body.style.overflow).toBe('')
    wrapper.unmount()
  })

  it('keeps the existing user popup dismissal behavior', async () => {
    const store = useAnnouncementStore()
    store.currentPopup = announcement
    const dismissPopup = vi.spyOn(store, 'dismissPopup').mockResolvedValue()
    const wrapper = mount(AnnouncementPopup)

    const dismissButton = document.body.querySelector<HTMLButtonElement>(
      '[data-testid="announcement-popup-dismiss"]',
    )
    dismissButton?.click()
    await wrapper.vm.$nextTick()

    expect(dismissPopup).toHaveBeenCalledTimes(1)
    expect(wrapper.emitted('close')).toBeUndefined()
    wrapper.unmount()
  })

  const popupItem = (id: number, readAt: string | null = null) => ({
    id,
    title: `Announcement ${id}`,
    content: 'body',
    notify_mode: 'popup' as const,
    read_at: readAt,
    created_at: '2026-07-24T07:30:00Z',
    updated_at: '2026-07-24T07:30:00Z',
  })

  it('offers mark-all-read when other unread popups remain', async () => {
    const store = useAnnouncementStore()
    const current = popupItem(1)
    store.currentPopup = current
    store.announcements = [current, popupItem(2), popupItem(3)] as never

    const wrapper = mount(AnnouncementPopup)
    await wrapper.vm.$nextTick()

    const markAll = document.body.querySelector<HTMLButtonElement>(
      '[data-testid="announcement-popup-mark-all-read"]',
    )
    expect(markAll).not.toBeNull()
    // 剩余未读数 = 2（不含当前这条）
    expect(document.body.textContent).toContain('announcements.unreadRemaining')

    wrapper.unmount()
  })

  it('hides mark-all-read when the current one is the only unread', async () => {
    const store = useAnnouncementStore()
    const current = popupItem(1)
    store.currentPopup = current
    store.announcements = [current, popupItem(2, '2026-07-24T08:00:00Z')] as never

    const wrapper = mount(AnnouncementPopup)
    await wrapper.vm.$nextTick()

    expect(
      document.body.querySelector('[data-testid="announcement-popup-mark-all-read"]'),
    ).toBeNull()

    wrapper.unmount()
  })

  it('marks everything read and lets the store close the popup when clicking mark-all-read', async () => {
    const store = useAnnouncementStore()
    const current = popupItem(1)
    store.currentPopup = current
    store.announcements = [current, popupItem(2)] as never
    // store.markAllAsRead 负责标记并关闭弹窗（清空队列），组件不再自行 dismiss
    const markAll = vi.spyOn(store, 'markAllAsRead').mockImplementation(async () => {
      store.announcements = store.announcements.map((a) => ({ ...a, read_at: '2026-07-24T09:00:00Z' })) as never
      store.currentPopup = null
    })
    const dismissPopup = vi.spyOn(store, 'dismissPopup').mockResolvedValue()

    const wrapper = mount(AnnouncementPopup)
    await wrapper.vm.$nextTick()

    document.body
      .querySelector<HTMLButtonElement>('[data-testid="announcement-popup-mark-all-read"]')
      ?.click()
    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(markAll).toHaveBeenCalledTimes(1)
    // 不能走 dismissPopup：那会把队列里的下一条又弹出来
    expect(dismissPopup).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('does not offer mark-all-read in preview mode', async () => {
    const store = useAnnouncementStore()
    store.currentPopup = popupItem(1)
    store.announcements = [popupItem(1), popupItem(2)] as never

    const wrapper = mount(AnnouncementPopup, { props: { announcement, preview: true } })
    await wrapper.vm.$nextTick()

    expect(
      document.body.querySelector('[data-testid="announcement-popup-mark-all-read"]'),
    ).toBeNull()

    wrapper.unmount()
  })
})
