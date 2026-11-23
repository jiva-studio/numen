import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate, SYNC_VIEW } from './index'
import { WORDS } from './words'

const CHECKOUT_URL = 'https://pay.example/checkout/abc'

interface Server {
  hasIdentity: boolean
}

const stubServer = (server: Server) => {
  const fetchStub = vi.fn((url: string | URL | Request) => {
    const method = String(url).split('/').at(-1)
    switch (method) {
      case 'GetStatus':
        return Promise.resolve(
          Response.json({ hasIdentity: server.hasIdentity, isReachable: true, items: [] }),
        )
      case 'GetSubscription':
        return Promise.resolve(
          Response.json({
            hasIdentity: server.hasIdentity,
            phase: 'none',
            periodEndAt: '',
            writesEndAt: '',
            deleteAt: '',
            plan: '',
            quotaBytes: 0,
            usedBytes: 0,
          }),
        )
      case 'ListPlans':
        return Promise.resolve(
          Response.json({
            plans: [
              { id: 'basic', quotaBytes: 10 * 2 ** 30 },
              { id: 'plus', quotaBytes: 100 * 2 ** 30 },
            ],
          }),
        )
      case 'CreateCheckout':
        return Promise.resolve(Response.json({ checkoutUrl: CHECKOUT_URL }))
      default:
        return Promise.resolve(
          Response.json({ code: 'internal', message: 'unknown' }, { status: 500 }),
        )
    }
  })
  vi.stubGlobal('fetch', fetchStub)
  return fetchStub
}

const setup = (server: Server = { hasIdentity: true }) => {
  stubServer(server)
  const context = createMockPluginContext({ pluginId: 'md.numen.tools.sync' })
  activate(context)
  const wrapper = mount(context.getRegisteredSlotViews()[0]?.component as never)
  return { context, wrapper }
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('the sync plugin frontend', () => {
  it('registers the sync status view', () => {
    const { context } = setup()

    expect(context.getRegisteredSlotViews().map((one) => one.id)).toStrictEqual([SYNC_VIEW])
    expect(SYNC_VIEW).toBe('sync')
  })

  it('shows synced in the title bar when the device has an identity, with no license', async () => {
    const { wrapper } = setup()

    await vi.advanceTimersByTimeAsync(0)

    expect(wrapper.find('.sync-badge__label').text()).toBe(WORDS.syncedLabel)
  })

  it('shows nothing in the title bar when the device has no identity', async () => {
    const { wrapper } = setup({ hasIdentity: false })

    await vi.advanceTimersByTimeAsync(0)

    expect(wrapper.find('.sync-badge').exists()).toBe(false)
  })

  it('follows the identity of the device every 10 seconds', async () => {
    const server: Server = { hasIdentity: true }
    const { wrapper } = setup(server)
    await vi.advanceTimersByTimeAsync(0)

    server.hasIdentity = false
    await vi.advanceTimersByTimeAsync(10_000)

    expect(wrapper.find('.sync-badge').exists()).toBe(false)
  })

  it('stops asking the backend when the plugin context is disposed', async () => {
    const server: Server = { hasIdentity: true }
    const { context, wrapper } = setup(server)
    await vi.advanceTimersByTimeAsync(0)

    context.dispose()
    server.hasIdentity = false
    await vi.advanceTimersByTimeAsync(60_000)

    expect(wrapper.find('.sync-badge__label').text()).toBe(WORDS.syncedLabel)
  })

  it('opens the payment page of the chosen plan through the identity of the plugin when the person buys', async () => {
    const { context } = setup()
    const card = context.getRegisteredSettingsViews().find((one) => one.id === 'subscription')
    const wrapper = mount(card?.component as never)
    await vi.advanceTimersByTimeAsync(0)
    await flushPromises()

    await wrapper
      .findAll('button')
      .find((one) => one.text() === 'Buy Plus, 100 GB')
      ?.trigger('click')
    await flushPromises()

    expect(context.getOpenedAddresses()).toStrictEqual([CHECKOUT_URL])
    wrapper.unmount()
  })
})
