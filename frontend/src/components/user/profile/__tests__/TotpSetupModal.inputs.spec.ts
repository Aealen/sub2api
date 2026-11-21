import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import TotpSetupModal from '../TotpSetupModal.vue'

const api = vi.hoisted(() => ({ getVerificationMethod: vi.fn(), initiateSetup: vi.fn(), enable: vi.fn() }))
vi.mock('@/api', () => ({ totpAPI: api }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('qrcode', () => ({ default: { toDataURL: vi.fn() } }))
enableAutoUnmount(afterEach)

beforeEach(() => {
  vi.resetAllMocks()
  api.getVerificationMethod.mockResolvedValue({ method: 'password' })
  api.initiateSetup.mockResolvedValue({ secret: 'EXAMPLE', qr_code_url: '', setup_token: 'setup-token' })
  api.enable.mockResolvedValue({ success: true })
})

// Error cascade drain (index * 20ms + 320ms) plus a small safety margin.
const DRAIN_MS = 600
// Success badge animation delays emit('success') by 450ms.
const SUCCESS_MS = 550

async function openCodeStep() {
  const wrapper = mount(TotpSetupModal, { attachTo: document.body })
  await flushPromises()
  await wrapper.get('input[type="password"]').setValue('password')
  await wrapper.get('.btn-primary').trigger('click')
  await flushPromises()
  await wrapper.get('.btn-primary').trigger('click')
  await wrapper.get('.cs-hidden-input').setValue('123456')
  return wrapper
}

const hiddenInput = (wrapper: ReturnType<typeof mount>) => wrapper.get('.cs-hidden-input')

describe('TOTP setup input synchronization', () => {
  it('clears displayed digits after verification fails and refocuses the hidden input', async () => {
    api.enable.mockRejectedValueOnce({ message: 'Invalid code' })
    const wrapper = await openCodeStep()
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect((hiddenInput(wrapper).element as HTMLInputElement).value).toBe('123456')
    // Wait for the error cascade drain to empty the slots.
    await new Promise(resolve => setTimeout(resolve, DRAIN_MS))
    expect((hiddenInput(wrapper).element as HTMLInputElement).value).toBe('')
    expect(document.activeElement).toBe(hiddenInput(wrapper).element)
    expect(wrapper.get('[type="submit"]').attributes('disabled')).toBeDefined()
    await hiddenInput(wrapper).setValue('654321')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    await new Promise(resolve => setTimeout(resolve, SUCCESS_MS))
    expect(api.enable).toHaveBeenLastCalledWith({ totp_code: '654321', setup_token: 'setup-token' })
    expect(wrapper.emitted('success')).toHaveLength(1)
  })

  it('displays retained digits when returning from the QR-code step', async () => {
    const wrapper = await openCodeStep()
    await wrapper.findAll('button').find(button => button.text() === 'common.back')!.trigger('click')
    await wrapper.get('.btn-primary').trigger('click')
    expect((hiddenInput(wrapper).element as HTMLInputElement).value).toBe('123456')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    await new Promise(resolve => setTimeout(resolve, SUCCESS_MS))
    expect(api.enable).toHaveBeenCalledWith({ totp_code: '123456', setup_token: 'setup-token' })
  })

  it('submits normally entered digits', async () => {
    const wrapper = await openCodeStep()
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    await new Promise(resolve => setTimeout(resolve, SUCCESS_MS))
    expect(api.enable).toHaveBeenCalledWith({ totp_code: '123456', setup_token: 'setup-token' })
    expect(wrapper.emitted('success')).toHaveLength(1)
  })
})
