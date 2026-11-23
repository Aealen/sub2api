import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import UpstreamsView from '../UpstreamsView.vue'
import UpstreamCompareView from '../UpstreamCompareView.vue'

const upstreamMocks = vi.hoisted(() => ({
  list: vi.fn(),
  unassignedAccounts: vi.fn(),
  getById: vi.fn(),
  compare: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(),
  assignAccounts: vi.fn(),
  upsertGroup: vi.fn(),
  deleteGroup: vi.fn(),
  upsertModelPrice: vi.fn(),
  deleteModelPrice: vi.fn(),
  importModelPrices: vi.fn(),
  importBundle: vi.fn()
}))

const storeMocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({ adminAPI: { upstreams: upstreamMocks } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => storeMocks }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const DataTableStub = defineComponent({
  name: 'DataTable',
  props: { columns: { type: Array, default: () => [] }, data: { type: Array, default: () => [] } },
  template: `
    <div data-testid="datatable">
      <div v-for="row in data" :key="row.id" class="dt-row">{{ row.name }}|{{ row.balance_total }}</div>
      <div v-if="data.length === 0" class="dt-empty"><slot name="empty" /></div>
    </div>
  `
})

const PassthroughStub = defineComponent({
  name: 'Passthrough',
  template: '<div><slot /><slot name="filters" /><slot name="table" /></div>'
})

const stubs = {
  AppLayout: PassthroughStub,
  TablePageLayout: PassthroughStub,
  DataTable: DataTableStub,
  Select: defineComponent({ name: 'Select', template: '<div class="select-stub"></div>' }),
  Icon: defineComponent({ name: 'Icon', template: '<i></i>' }),
  ConfirmDialog: defineComponent({ name: 'ConfirmDialog', template: '<div></div>' }),
  'router-link': defineComponent({ name: 'RouterLink', template: '<a><slot /></a>' }),
  UpstreamFormDialog: true,
  UpstreamDetailDialog: true,
  UnassignedAccountsDialog: true
}

beforeEach(() => {
  vi.clearAllMocks()
  upstreamMocks.unassignedAccounts.mockResolvedValue({ items: [] })
})

describe('UpstreamsView', () => {
  it('renders upstream rows with balance aggregation', async () => {
    upstreamMocks.list.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'PPDog',
          base_url: 'https://api.pp.dog',
          website: '',
          contact: '',
          note: '',
          status: 'active',
          created_at: '',
          updated_at: '',
          account_total: 2,
          account_active: 2,
          balance_total: 57.28,
          balance_updated_at: null,
          rate_multiplier_min: 0.12,
          rate_multiplier_max: 0.12,
          model_price_count: 0,
          group_count: 0,
          group_names: []
        }
      ],
      total: 1
    })

    const wrapper = mount(UpstreamsView, { global: { stubs } })
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('PPDog')
    expect(text).toContain('57.28')
    expect(wrapper.findAll('.dt-row')).toHaveLength(1)
  })

  it('shows the empty state when no upstream exists', async () => {
    upstreamMocks.list.mockResolvedValue({ items: [], total: 0 })

    const wrapper = mount(UpstreamsView, { global: { stubs } })
    await flushPromises()

    expect(wrapper.find('.dt-empty').exists()).toBe(true)
  })
})

describe('UpstreamCompareView', () => {
  it('renders a row even when the backend returns null cells', async () => {
    upstreamMocks.compare.mockResolvedValue({
      upstreams: [{ id: 1, name: 'Hualong' }],
      columns: [
        { upstream_id: 1, upstream_name: 'Hualong', group_id: 100, group_name: 'default', rate_multiplier: 0.48, kind: 'group' }
      ],
      rows: [
        {
          model_name: 'claude-sonnet-4-5',
          platform: 'anthropic',
          official_input_price_per_token: 3e-6,
          official_output_price_per_token: 1.5e-5,
          // 后端 nil slice 曾被序列化成 null：这里显式回归
          cells: null,
          cheapest_upstream_id: null,
          cheapest_group_id: null
        }
      ]
    })

    const wrapper = mount(UpstreamCompareView, { global: { stubs } })
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('claude-sonnet-4-5')
    expect(text).toContain('Hualong')
  })

  it('marks ledger-sourced cells and shows the rate multiplier', async () => {
    upstreamMocks.compare.mockResolvedValue({
      upstreams: [{ id: 1, name: 'Hualong' }],
      columns: [
        { upstream_id: 1, upstream_name: 'Hualong', group_id: 100, group_name: 'default', rate_multiplier: 0.48, kind: 'group' },
        { upstream_id: 1, upstream_name: 'Hualong', group_id: null, group_name: '综合', rate_multiplier: null, kind: 'derived' }
      ],
      rows: [
        {
          model_name: 'deepseek-v4-pro',
          platform: 'deepseek',
          official_input_price_per_token: 6.6e-7,
          official_output_price_per_token: 1.98e-6,
          cells: [
            {
              upstream_id: 1,
              group_id: 100,
              upstream_name: 'Hualong',
              input_price_per_token: 2.16e-7,
              output_price_per_token: null,
              rate_multiplier: 0.48,
              source: 'ledger'
            }
          ],
          cheapest_upstream_id: null,
          cheapest_group_id: null
        }
      ]
    })

    const wrapper = mount(UpstreamCompareView, { global: { stubs } })
    await flushPromises()

    const cell = wrapper.find('[data-testid="upstream-compare-cell-1:100"]')
    expect(cell.exists()).toBe(true)
    expect(cell.text()).toContain('0.2160')
    expect(cell.text()).toContain('0.48')
    expect(cell.text()).toContain('admin.upstreams.compare.sourceLedger')
    // 二级表头：第一级供应商 colspan、第二级分组名
    expect(wrapper.text()).toContain('Hualong')
    expect(wrapper.text()).toContain('default')
  })
})
