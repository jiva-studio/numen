import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
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

const render = async (response: SubscriptionResponse) => {
  const client = createClient(() => Promise.resolve(response))
  const openExternal = vi.fn<(url: string) => Promise<Result<null>>>(() =>
    Promise.resolve(asValue(null)),
  )
  const wrapper = mount(SubscriptionCard, { props: { client, openExternal, now: () => NOW } })
  await flushPromises()
  return { wrapper, client, openExternal }
}

const findBuy = (wrapper: VueWrapper, label?: string) =>
  wrapper
    .findAll('button')
    .find((one) =>
      label === undefined ? one.text().startsWith('Buy ') : one.text() === `Buy ${label}`,
    )

const findChange = (wrapper: VueWrapper, name?: string) =>
  wrapper
    .findAll('button')
    .find((one) =>
      name === undefined
        ? one.text().startsWith('Change plan to ')
        : one.text() === `Change plan to ${name}`,
    )

describe('SubscriptionCard', () => {
  it('shows the title Numen Sync', async () => {
    const { wrapper } = await render(createResponse())

    expect(wrapper.text()).toContain('Numen Sync')
  })

  it('says there is no subscription and offers to buy it', async () => {
    const { wrapper } = await render(createResponse({ phase: 'none' }))

    expect(wrapper.text()).toContain('No subscription')
    expect(wrapper.text()).toContain('Buy a subscription to sync your vaults between your devices.')
    expect(findBuy(wrapper)).toBeDefined()
  })

  it('shows the end of the paid period of an active subscription and offers nothing', async () => {
    const { wrapper } = await render(
      createResponse({ phase: 'active', periodEndAt: '2026-12-23T12:00:00Z' }),
    )

    expect(wrapper.text()).toContain('Active')
    expect(wrapper.text()).toContain('Active until 23 Dec 2026.')
    expect(findBuy(wrapper)).toBeUndefined()
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
      'The subscription ended on 20 Nov 2026. Changes are sent until 27 Nov 2026, then the vaults in the cloud can only be read.',
    )
    expect(findBuy(wrapper)).toBeDefined()
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
      'The subscription ended on 13 Nov 2026. The vaults in the cloud can only be read and changes stay on this device. The cloud deletes them on 19 Jan 2027.',
    )
    expect(findBuy(wrapper)).toBeDefined()
  })

  it('says the cloud deleted the vaults and offers to buy', async () => {
    const { wrapper } = await render(createResponse({ phase: 'deleted' }))

    expect(wrapper.text()).toContain('Deleted')
    expect(wrapper.text()).toContain(
      'The cloud deleted your vaults. Buy a subscription to sync again.',
    )
    expect(findBuy(wrapper)).toBeDefined()
  })

  it('asks for an identity and offers nothing when the device has none', async () => {
    const { wrapper } = await render(createResponse({ hasIdentity: false, phase: '' }))

    expect(wrapper.text()).toContain('No identity')
    expect(wrapper.text()).toContain('Create an identity to buy a subscription.')
    expect(findBuy(wrapper)).toBeUndefined()
  })

  it('says the server cannot be reached and offers nothing', async () => {
    const client = createClient(() =>
      Promise.reject(
        Object.assign(new Error('Could not reach the sync server'), { code: 'unavailable' }),
      ),
    )
    const wrapper = mount(SubscriptionCard, {
      props: { client, openExternal: vi.fn(() => Promise.resolve(asValue(null))), now: () => NOW },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Could not reach the sync server')
    expect(findBuy(wrapper)).toBeUndefined()
  })

  it('opens the payment page in the browser and waits when the person buys', async () => {
    const { wrapper, client, openExternal } = await render(createResponse({ phase: 'none' }))

    await findBuy(wrapper, 'Plus, 100 GB, $15')?.trigger('click')
    await flushPromises()

    expect(client.createCheckout).toHaveBeenCalledExactlyOnceWith('plus')
    expect(openExternal).toHaveBeenCalledExactlyOnceWith(CHECKOUT_URL)
    const waiting = wrapper
      .findAll('button')
      .find((one) => one.text() === 'Waiting for the payment…')
    expect(waiting?.attributes('disabled')).toBeDefined()
    expect(findBuy(wrapper)).toBeUndefined()
  })

  it('offers to buy every plan with its name and size when there is no subscription', async () => {
    const { wrapper } = await render(createResponse({ phase: 'none' }))

    expect(findBuy(wrapper, 'Basic, 10 GB, $5')).toBeDefined()
    expect(findBuy(wrapper, 'Plus, 100 GB, $15')).toBeDefined()
    expect(findBuy(wrapper, 'Pro, 250 GB, $30')).toBeDefined()
    expect(findChange(wrapper)).toBeUndefined()
  })

  it('shows the plan with its size and the space used while the subscription is active', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    expect(wrapper.text()).toContain('Plus, 100 GB')
    expect(wrapper.text()).toContain('2 GB of 100 GB used')
    expect(findBuy(wrapper)).toBeUndefined()
  })

  it('offers to change to every other plan and not to the plan the account has', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    expect(findChange(wrapper, 'Basic')).toBeDefined()
    expect(findChange(wrapper, 'Pro')).toBeDefined()
    expect(findChange(wrapper, 'Plus')).toBeUndefined()
  })

  it('changes the plan without opening a browser and shows the new plan', async () => {
    const { wrapper, client, openExternal } = await render(ACTIVE_ON_PLUS)

    await findChange(wrapper, 'Pro')?.trigger('click')
    await flushPromises()

    expect(client.changePlan).toHaveBeenCalledExactlyOnceWith('pro')
    expect(openExternal).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('Pro, 250 GB, $30')
    expect(findChange(wrapper, 'Pro')).toBeUndefined()
    expect(findChange(wrapper, 'Plus')).toBeDefined()
  })

  it('shows the problem of a refused change and keeps the plan', async () => {
    const { wrapper, client } = await render(ACTIVE_ON_PLUS)
    client.changePlan.mockRejectedValue(
      Object.assign(new Error('Could not reach the sync server'), { code: 'unavailable' }),
    )

    await findChange(wrapper, 'Pro')?.trigger('click')
    await flushPromises()

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
    expect(wrapper.text()).toContain('Cloud storage is full: changes stay on this device')
  })

  it('does not say the cloud is full while there is room', async () => {
    const { wrapper } = await render(ACTIVE_ON_PLUS)

    expect(wrapper.text()).not.toContain('Cloud storage is full')
  })

  it('shows the plan of an ended subscription with the offers to buy and none to change', async () => {
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
    expect(findBuy(wrapper, 'Plus, 100 GB, $15')).toBeDefined()
    expect(findChange(wrapper)).toBeUndefined()
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

    expect(wrapper.text()).toContain('Trial until 7 Dec 2026.')
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
    const wrapper = mount(SubscriptionCard, { props: { client, openExternal, now: () => NOW } })
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

describe('SubscriptionCard products', () => {
  const PRODUCTS = [
    { id: 'price-extra', slug: 'extra', name: 'Extra', priceCents: 499, currency: 'EUR', oneTime: true },
    { id: 'price-theme', slug: 'theme', name: 'Theme', priceCents: 199, currency: 'EUR', oneTime: true },
  ] as const

  const renderProducts = async (owned: readonly string[]) => {
    const client = {
      ...createClient(() => Promise.resolve(createResponse())),
      listProducts: vi.fn<NonNullable<SubscriptionClient['listProducts']>>(() =>
        Promise.resolve(PRODUCTS),
      ),
      getOwnedProducts: vi.fn<NonNullable<SubscriptionClient['getOwnedProducts']>>(() =>
        Promise.resolve(owned),
      ),
    }
    const openExternal = vi.fn<(url: string) => Promise<Result<null>>>(() =>
      Promise.resolve(asValue(null)),
    )
    const wrapper = mount(SubscriptionCard, { props: { client, openExternal, now: () => NOW } })
    await flushPromises()
    return { wrapper, client, openExternal }
  }

  it('marks an owned product Owned and offers to buy the other one', async () => {
    const { wrapper } = await renderProducts(['extra'])

    const list = wrapper.find('ul[aria-label="Products"]')
    expect(list.text()).toContain('Extra')
    expect(list.text()).toContain('Owned')
    expect(list.text()).not.toContain('Theme')
    expect(findBuy(wrapper, 'Theme, €1.99')).toBeDefined()
    expect(findBuy(wrapper, 'Extra, €4.99')).toBeUndefined()
  })

  it('opens the payment page of the price of the product when the person buys it', async () => {
    const { wrapper, client, openExternal } = await renderProducts([])

    await findBuy(wrapper, 'Extra, €4.99')?.trigger('click')
    await flushPromises()

    expect(client.createCheckout).toHaveBeenCalledExactlyOnceWith('price-extra')
    expect(openExternal).toHaveBeenCalledExactlyOnceWith(CHECKOUT_URL)
  })
})
