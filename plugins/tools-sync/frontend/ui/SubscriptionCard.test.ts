import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { asValue, type Failure, type OpenExternalErrorCode, type Result } from '@numen/plugin-sdk'
import type { SubscriptionClient } from '../api/subscription'
import SubscriptionCard from './SubscriptionCard.vue'

type SubscriptionResponse = Awaited<ReturnType<SubscriptionClient['getSubscription']>>

const NOW = new Date('2026-11-23T12:00:00Z')
const CHECKOUT_URL = 'https://pay.example/checkout/abc'

const createResponse = (patch: Partial<SubscriptionResponse> = {}): SubscriptionResponse => ({
  hasIdentity: true,
  phase: 'none',
  periodEndAt: '',
  writesEndAt: '',
  deleteAt: '',
  ...patch,
})

const createClient = (answer: () => Promise<SubscriptionResponse>) => ({
  getSubscription: vi.fn<SubscriptionClient['getSubscription']>(answer),
  createCheckout: vi.fn<SubscriptionClient['createCheckout']>(() => Promise.resolve(CHECKOUT_URL)),
})

const render = async (response: SubscriptionResponse) => {
  const client = createClient(() => Promise.resolve(response))
  const openExternal = vi.fn<
    (url: string) => Promise<Result<null, Failure<OpenExternalErrorCode>>>
  >(() => Promise.resolve(asValue(null)))
  const wrapper = mount(SubscriptionCard, { props: { client, openExternal, now: () => NOW } })
  await flushPromises()
  return { wrapper, client, openExternal }
}

const findBuy = (wrapper: VueWrapper) =>
  wrapper.findAll('button').find((one) => one.text() === 'Buy')

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

    await findBuy(wrapper)?.trigger('click')
    await flushPromises()

    expect(client.createCheckout).toHaveBeenCalledOnce()
    expect(openExternal).toHaveBeenCalledExactlyOnceWith(CHECKOUT_URL)
    const waiting = wrapper
      .findAll('button')
      .find((one) => one.text() === 'Waiting for the payment…')
    expect(waiting?.attributes('disabled')).toBeDefined()
    expect(findBuy(wrapper)).toBeUndefined()
  })
})
