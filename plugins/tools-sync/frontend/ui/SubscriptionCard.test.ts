import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { asValue, type Result } from '@numen/plugin-sdk'
import type { SubscriptionClient } from '../api/subscription'
import SubscriptionCard from './SubscriptionCard.vue'

type SubscriptionResponse = Awaited<ReturnType<SubscriptionClient['getSubscription']>>

const NOW = new Date('2026-11-23T12:00:00Z')
const CHECKOUT_URL = 'https://pay.example/checkout/abc'

const GIB = 2 ** 30
const PLANS = [
  { id: 'basic', name: 'Basic', quotaBytes: 10 * GIB, priceCents: 500, currency: 'USD' },
  { id: 'plus', name: 'Plus', quotaBytes: 100 * GIB, priceCents: 1500, currency: 'USD' },
  { id: 'pro', name: 'Pro', quotaBytes: 250 * GIB, priceCents: 3000, currency: 'USD' },
] as const
const quotaOf = (plan: string): number => PLANS.find((one) => one.id === plan)?.quotaBytes ?? 0

const createResponse = (patch: Partial<SubscriptionResponse> = {}): SubscriptionResponse => ({
  hasIdentity: true,
  phase: 'none',
  periodEndAt: '',
  writesEndAt: '',
  deleteAt: '',
  plan: '',
  quotaBytes: 0,
  usedBytes: 0,
  ...patch,
})

const ACTIVE_ON_PLUS = createResponse({
  phase: 'active',
  periodEndAt: '2026-12-23T12:00:00Z',
  plan: 'plus',
  quotaBytes: 100 * GIB,
  usedBytes: 2 * GIB,
})

const createClient = (answer: () => Promise<SubscriptionResponse>) => ({
  getSubscription: vi.fn<SubscriptionClient['getSubscription']>(answer),
  listPlans: vi.fn<SubscriptionClient['listPlans']>(() => Promise.resolve(PLANS)),
  changePlan: vi.fn<SubscriptionClient['changePlan']>((plan) =>
    Promise.resolve(createResponse({ ...ACTIVE_ON_PLUS, plan, quotaBytes: quotaOf(plan) })),
  ),
  createCheckout: vi.fn<SubscriptionClient['createCheckout']>(() => Promise.resolve(CHECKOUT_URL)),
})

const mounted: VueWrapper[] = []

afterEach(() => {
  mounted.splice(0).forEach((one) => one.unmount())
})

const mountCard = (props: InstanceType<typeof SubscriptionCard>['$props']) => {
  const wrapper = mount(SubscriptionCard, { props, attachTo: document.body })
  mounted.push(wrapper)
  return wrapper
}

const render = async (response: SubscriptionResponse) => {
  const client = createClient(() => Promise.resolve(response))
  const openExternal = vi.fn<(url: string) => Promise<Result<null>>>(() =>
    Promise.resolve(asValue(null)),
  )
  const wrapper = mountCard({ client, openExternal, now: () => NOW })
  await flushPromises()
  return { wrapper, client, openExternal }
}

const findChoose = (wrapper: VueWrapper, name?: string) =>
  wrapper
    .findAll('button')
    .find((one) =>
      name === undefined
        ? (one.attributes('aria-label') ?? '').startsWith('Choose ')
        : one.attributes('aria-label') === `Choose ${name}`,
    )

const findPlanSelect = (wrapper: VueWrapper) => wrapper.find('button[data-slot="select"]')

const getPlanChoices = () =>
  Array.from(document.body.querySelectorAll<HTMLElement>('.menu__item')).map(
    (one) => one.querySelector('.menu__text')?.textContent?.trim() ?? '',
  )

const changePlanTo = async (wrapper: VueWrapper, name: string) => {
  await findPlanSelect(wrapper).trigger('click')
  await flushPromises()
  const choice = Array.from(document.body.querySelectorAll<HTMLElement>('.menu__item')).find(
    (one) => one.querySelector('.menu__text')?.textContent?.trim() === name,
  )
  choice?.click()
  await flushPromises()
}

describe('SubscriptionCard', () => {
  it('shows the title Numen Sync', async () => {
    const { wrapper } = await render(createResponse())

    expect(wrapper.text()).toContain('Numen Sync')
  })

  it('offers the plans to choose and shows no badge when there is no subscription', async () => {
    const { wrapper } = await render(createResponse({ phase: 'none' }))

    expect(wrapper.find('.subscription__badge').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('No subscription')
    expect(wrapper.text()).toContain('Buy a subscription to sync your vaults between your devices.')
    expect(findChoose(wrapper)).toBeDefined()
  })

  it('names the plan in the badge and says until when the active subscription is paid', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    expect(wrapper.get('.subscription__badge').text()).toBe('Plus')
    expect(wrapper.get('.subscription__facts').text()).toContain('Active until')
    expect(wrapper.get('.subscription__facts').text()).toContain('December 23, 2026')
    expect(findChoose(wrapper)).toBeUndefined()
  })

  it('shows the plan, the space used as a bar and the end of the paid period as rows', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    const rows = wrapper.get('.subscription__facts').text()
    expect(rows).toContain('Plan')
    expect(rows).toContain('Plus, 100 GB')
    expect(rows).toContain('Storage')
    expect(rows).toContain('2 GB of 100 GB used')
    expect(wrapper.get('[role="progressbar"]').attributes('aria-valuenow')).toBe('2')
  })

  it('says when the changes stop in the grace phase and offers to buy', async () => {
    const { wrapper } = await render(
      createResponse({
        phase: 'grace',
        periodEndAt: '2026-11-20T12:00:00Z',
        writesEndAt: '2026-11-27T12:00:00Z',
      }),
    )

    expect(wrapper.text()).toContain('Ended')
    expect(wrapper.text()).toContain(
      'The subscription ended on November 20, 2026. Changes are sent until November 27, 2026, then the vaults in the cloud can only be read.',
    )
    expect(findChoose(wrapper)).toBeDefined()
  })

  it('says the vaults can only be read and when the cloud deletes them, and offers to buy', async () => {
    const { wrapper } = await render(
      createResponse({
        phase: 'readOnly',
        periodEndAt: '2026-11-13T12:00:00Z',
        writesEndAt: '2026-11-20T12:00:00Z',
        deleteAt: '2027-01-19T12:00:00Z',
      }),
    )

    expect(wrapper.text()).toContain('Read only')
    expect(wrapper.text()).toContain(
      'The subscription ended on November 13, 2026. The vaults in the cloud can only be read and changes stay on this device. The cloud deletes them on January 19, 2027.',
    )
    expect(findChoose(wrapper)).toBeDefined()
  })

  it('says the cloud deleted the vaults and offers to buy', async () => {
    const { wrapper } = await render(createResponse({ phase: 'deleted' }))

    expect(wrapper.text()).toContain('Deleted')
    expect(wrapper.text()).toContain(
      'The cloud deleted your vaults. Buy a subscription to sync again.',
    )
    expect(findChoose(wrapper)).toBeDefined()
  })

  it('asks for an identity and offers nothing when the device has none', async () => {
    const { wrapper } = await render(createResponse({ hasIdentity: false, phase: '' }))

    expect(wrapper.text()).toContain('Create an identity to buy a subscription.')
    expect(wrapper.find('.subscription__badge').exists()).toBe(false)
    expect(findChoose(wrapper)).toBeUndefined()
  })

  it('says the server cannot be reached and offers nothing', async () => {
    const client = createClient(() =>
      Promise.reject(
        Object.assign(new Error('Could not reach the sync server'), { code: 'unavailable' }),
      ),
    )
    const wrapper = mountCard({
      client,
      openExternal: vi.fn(() => Promise.resolve(asValue(null))),
      now: () => NOW,
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Could not reach the sync server')
    expect(findChoose(wrapper)).toBeUndefined()
  })

  it('opens the payment page in the browser and waits when the person buys', async () => {
    const { wrapper, client, openExternal } = await render(createResponse({ phase: 'none' }))

    await findChoose(wrapper, 'Plus')?.trigger('click')
    await flushPromises()

    expect(client.createCheckout).toHaveBeenCalledExactlyOnceWith('plus')
    expect(openExternal).toHaveBeenCalledExactlyOnceWith(CHECKOUT_URL)
    expect(wrapper.get('[role="status"]').text()).toContain('Waiting for the payment…')
    expect(wrapper.text()).not.toContain('Choose a plan')
    expect(findChoose(wrapper)).toBeUndefined()
  })

  it('asks to choose a plan above the plans when there is no subscription', async () => {
    const { wrapper } = await render(createResponse({ phase: 'none' }))

    expect(wrapper.text()).toContain('Choose a plan')
  })

  it('lays the plans out as tiles with the name, the size, the price and the button Choose', async () => {
    const { wrapper } = await render(createResponse({ phase: 'none' }))

    const tiles = wrapper.findAll('.subscription__tile')
    const expected = [
      ['Basic', '10 GB', '$5 / month'],
      ['Plus', '100 GB', '$15 / month'],
      ['Pro', '250 GB', '$30 / month'],
    ]
    expect(tiles).toHaveLength(expected.length)
    tiles.forEach((tile, at) => {
      ;[...(expected[at] ?? []), 'Choose'].forEach((part) => expect(tile.text()).toContain(part))
    })
  })

  it('shows the plans by itself when the identity is created while the card is open', async () => {
    let answer = createResponse({ hasIdentity: false, phase: '' })
    const client = createClient(() => Promise.resolve(answer))
    const wrapper = mountCard({
      client,
      openExternal: vi.fn(() => Promise.resolve(asValue(null))),
      now: () => NOW,
      refreshMs: { noIdentity: 50, ready: 60_000, unreachable: 60_000 },
    })
    await flushPromises()
    expect(wrapper.text()).toContain('Create an identity to buy a subscription.')

    answer = createResponse({ phase: 'none' })
    await vi.waitFor(() => expect(findChoose(wrapper, 'Plus')).toBeDefined())

    expect(wrapper.text()).not.toContain('Create an identity')
  })

  it('says which server it could not reach and opens the settings from the card', async () => {
    const client = createClient(() =>
      Promise.reject(
        Object.assign(new Error('Could not reach the sync server'), { code: 'unavailable' }),
      ),
    )
    const openSettings = vi.fn<() => void>()
    const wrapper = mountCard({
      client,
      openExternal: vi.fn(() => Promise.resolve(asValue(null))),
      now: () => NOW,
      settings: {
        get: <T>() => 'https://sync.numen.md' as T,
        onChange: () => ({ dispose: () => undefined }),
      },
      openSettings,
      refreshMs: { noIdentity: 60_000, ready: 60_000, unreachable: 60_000 },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('https://sync.numen.md')
    expect(wrapper.text()).toContain('Check that it is running')
    expect(wrapper.text()).not.toContain('30 seconds')
    await wrapper
      .findAll('button')
      .find((one) => one.text() === 'Open the settings')
      ?.trigger('click')
    expect(openSettings).toHaveBeenCalledOnce()
  })

  it('offers every plan to choose with its name when there is no subscription', async () => {
    const { wrapper } = await render(createResponse({ phase: 'none' }))

    expect(findChoose(wrapper, 'Basic')).toBeDefined()
    expect(findChoose(wrapper, 'Plus')).toBeDefined()
    expect(findChoose(wrapper, 'Pro')).toBeDefined()
    expect(findPlanSelect(wrapper).exists()).toBe(false)
  })

  it('offers one menu Change plan with every other plan and not the plan the account has', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    expect(findPlanSelect(wrapper).text()).toBe('Change plan')
    await findPlanSelect(wrapper).trigger('click')
    await flushPromises()

    expect(getPlanChoices()).toStrictEqual(['Basic', 'Pro'])
  })

  it('offers one Manage subscription and no row of plan buttons while the subscription is active', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    expect(wrapper.findAll('.subscription__actions button')).toHaveLength(2)
    expect(wrapper.findAll('.subscription__tile')).toHaveLength(0)
  })

  it('changes the plan without opening a browser and shows the new plan', async () => {
    const { wrapper, client, openExternal } = await render(ACTIVE_ON_PLUS)

    await changePlanTo(wrapper, 'Pro')

    expect(client.changePlan).toHaveBeenCalledExactlyOnceWith('pro')
    expect(openExternal).not.toHaveBeenCalled()
    expect(wrapper.get('.subscription__badge').text()).toBe('Pro')
    expect(wrapper.get('.subscription__facts').text()).toContain('Pro, 250 GB')
    expect(findPlanSelect(wrapper).text()).toBe('Change plan')
  })

  it('shows the problem of a refused change and keeps the plan', async () => {
    const { wrapper, client } = await render(ACTIVE_ON_PLUS)
    client.changePlan.mockRejectedValue(
      Object.assign(new Error('Could not reach the sync server'), { code: 'unavailable' }),
    )

    await changePlanTo(wrapper, 'Pro')

    expect(wrapper.find('[role="alert"]').text()).toContain('Could not reach the sync server')
    expect(wrapper.text()).toContain('Plus, 100 GB')
  })

  it('shows the used space above the plan and says the cloud is full', async () => {
    const { wrapper } = await render(
      createResponse({
        ...ACTIVE_ON_PLUS,
        plan: 'basic',
        quotaBytes: 10 * GIB,
        usedBytes: 20 * GIB,
      }),
    )

    expect(wrapper.text()).toContain('20 GB of 10 GB used')
    expect(wrapper.get('[role="progressbar"]').attributes('aria-valuenow')).toBe('100')
    expect(wrapper.text()).toContain('Cloud storage is full: changes stay on this device')
  })

  it('does not say the cloud is full while there is room', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    expect(wrapper.text()).not.toContain('Cloud storage is full')
  })

  it('shows the plan of an ended subscription with the plans to choose and none to change', async () => {
    const { wrapper } = await render(
      createResponse({
        phase: 'grace',
        periodEndAt: '2026-11-20T12:00:00Z',
        writesEndAt: '2026-11-27T12:00:00Z',
        plan: 'plus',
        quotaBytes: 100 * GIB,
      }),
    )

    expect(wrapper.text()).toContain('Plus, 100 GB')
    expect(findChoose(wrapper, 'Plus')).toBeDefined()
    expect(findPlanSelect(wrapper).exists()).toBe(false)
  })

  it('shows only the size of a plan the client does not know', async () => {
    const { wrapper } = await render(
      createResponse({ ...ACTIVE_ON_PLUS, plan: 'gold', quotaBytes: 50 * GIB }),
    )

    expect(wrapper.text()).toContain('50 GB')
    expect(wrapper.text()).not.toContain('gold')
  })
})

describe('SubscriptionCard trial and portal', () => {
  const PORTAL_URL = 'https://portal.example/session/1'

  it('says the day the trial ends', async () => {
    const { wrapper } = await render(
      createResponse({ ...ACTIVE_ON_PLUS, isTrial: true, periodEndAt: '2026-12-07T12:00:00Z' }),
    )

    expect(wrapper.get('.subscription__facts').text()).toContain('Trial until')
    expect(wrapper.get('.subscription__facts').text()).toContain('December 7, 2026')
    expect(wrapper.text()).not.toContain('Active until')
  })

  it('opens the portal in the browser with the one button Manage subscription', async () => {
    const client = {
      ...createClient(() => Promise.resolve(ACTIVE_ON_PLUS)),
      createPortalSession: vi.fn<() => Promise<string>>(() => Promise.resolve(PORTAL_URL)),
    }
    const openExternal = vi.fn<(url: string) => Promise<Result<null>>>(() =>
      Promise.resolve(asValue(null)),
    )
    const wrapper = mountCard({ client, openExternal, now: () => NOW })
    await flushPromises()

    const manage = wrapper.findAll('button').filter((one) => one.text() === 'Manage subscription')
    await manage[0]?.trigger('click')
    await flushPromises()

    expect(manage).toHaveLength(1)
    expect(openExternal).toHaveBeenCalledExactlyOnceWith(PORTAL_URL)
  })

  it('offers no portal to an account without a subscription', async () => {
    const { wrapper } = await render(createResponse())

    expect(wrapper.text()).not.toContain('Manage subscription')
  })
})
