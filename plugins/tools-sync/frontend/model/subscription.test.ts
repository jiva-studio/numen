import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { asFailure, asValue, type Result } from '@numen/plugin-sdk'
import type { SubscriptionClient } from '../api/subscription'
import { useSubscription } from './subscription'

type SubscriptionResponse = Awaited<ReturnType<SubscriptionClient['getSubscription']>>

const NOW = new Date('2026-11-23T12:00:00Z')
const CHECKOUT_URL = 'https://pay.example/checkout/abc'
const NOT_SEEN = 'The payment is not seen yet. Open this page again after you pay.'

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

const ACTIVE = createResponse({
  phase: 'active',
  periodEndAt: '2026-12-23T12:00:00Z',
  writesEndAt: '2026-12-30T12:00:00Z',
  deleteAt: '2027-02-28T12:00:00Z',
})

const GIB = 2 ** 30
const PLANS = [
  { id: 'basic', quotaBytes: 10 * GIB },
  { id: 'plus', quotaBytes: 100 * GIB },
  { id: 'pro', quotaBytes: 250 * GIB },
] as const
const quotaOf = (plan: string): number => PLANS.find((one) => one.id === plan)?.quotaBytes ?? 0

const ACTIVE_ON_PLUS = createResponse({
  phase: 'active',
  periodEndAt: '2026-12-23T12:00:00Z',
  plan: 'plus',
  quotaBytes: 100 * GIB,
  usedBytes: 2 * GIB,
})

const createFailure = (code: string, message: string): Error =>
  Object.assign(new Error(message), { code })

const createClient = (answers: readonly SubscriptionResponse[]) => {
  let read = 0
  return {
    getSubscription: vi.fn<SubscriptionClient['getSubscription']>(() => {
      const answer = answers.at(Math.min(read, answers.length - 1))
      read += 1
      return Promise.resolve(answer ?? createResponse())
    }),
    listPlans: vi.fn<SubscriptionClient['listPlans']>(() => Promise.resolve(PLANS)),
    changePlan: vi.fn<SubscriptionClient['changePlan']>((plan) =>
      Promise.resolve(createResponse({ ...ACTIVE_ON_PLUS, plan, quotaBytes: quotaOf(plan) })),
    ),
    createCheckout: vi.fn<SubscriptionClient['createCheckout']>(() =>
      Promise.resolve(CHECKOUT_URL),
    ),
  }
}

type OpenExternal = (url: string) => Promise<Result<null>>

const createOpen = () => vi.fn<OpenExternal>(() => Promise.resolve(asValue(null)))

const mountSubscription = (
  client: ReturnType<typeof createClient>,
  openExternal: ReturnType<typeof createOpen> = createOpen(),
) => {
  const state: { current?: ReturnType<typeof useSubscription> } = {}
  const wrapper = mount(
    defineComponent({
      setup() {
        state.current = useSubscription({ client, openExternal, now: () => NOW })
        return () => h('div')
      },
    }),
  )
  const subscription = state.current
  if (!subscription) throw new Error('the composable did not run')
  return { subscription, wrapper, openExternal }
}

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('useSubscription on load', () => {
  it('is loading until the server answers', async () => {
    const { subscription } = mountSubscription(createClient([createResponse()]))

    expect(subscription.status.value).toBe('loading')
    await subscription.load()
  })

  it('is ready with the phase none and offers to buy for an account without a subscription', async () => {
    const { subscription } = mountSubscription(createClient([createResponse()]))

    await subscription.load()

    expect(subscription.status.value).toBe('ready')
    expect(subscription.phase.value).toBe('none')
    expect(subscription.canBuy.value).toBe(true)
  })

  it('is ready with the phase active and does not offer to buy', async () => {
    const { subscription } = mountSubscription(createClient([ACTIVE]))

    await subscription.load()

    expect(subscription.phase.value).toBe('active')
    expect(subscription.canBuy.value).toBe(false)
  })

  it('offers to buy in each phase that is not active', async () => {
    for (const phase of ['none', 'grace', 'readOnly', 'deleted'] as const) {
      const { subscription } = mountSubscription(createClient([createResponse({ phase })]))

      await subscription.load()

      expect(subscription.canBuy.value, phase).toBe(true)
    }
  })

  it('is noIdentity and offers nothing when the device has no identity', async () => {
    const { subscription } = mountSubscription(
      createClient([createResponse({ hasIdentity: false, phase: '' })]),
    )

    await subscription.load()

    expect(subscription.status.value).toBe('noIdentity')
    expect(subscription.canBuy.value).toBe(false)
  })

  it('is unreachable and offers nothing when the server does not answer', async () => {
    const client = createClient([createResponse()])
    client.getSubscription.mockRejectedValue(
      createFailure('unavailable', 'Could not reach the sync server'),
    )
    const { subscription } = mountSubscription(client)

    await subscription.load()

    expect(subscription.status.value).toBe('unreachable')
    expect(subscription.canBuy.value).toBe(false)
  })
})

describe('useSubscription buy', () => {
  it('opens the payment page in the system browser and waits for the payment', async () => {
    const { subscription, openExternal } = mountSubscription(createClient([createResponse()]))
    await subscription.load()

    await subscription.buy('plus')

    expect(openExternal).toHaveBeenCalledExactlyOnceWith(CHECKOUT_URL)
    expect(subscription.waiting.value).toBe(true)
  })

  it('asks the server again every 3 seconds and ends when the phase is active', async () => {
    const client = createClient([createResponse(), createResponse(), createResponse(), ACTIVE])
    const { subscription } = mountSubscription(client)
    await subscription.load()
    await subscription.buy('plus')
    expect(client.getSubscription).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(2999)
    expect(client.getSubscription).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1)
    expect(client.getSubscription).toHaveBeenCalledTimes(2)
    await vi.advanceTimersByTimeAsync(6000)

    expect(subscription.phase.value).toBe('active')
    expect(subscription.waiting.value).toBe(false)
    const reads = client.getSubscription.mock.calls.length
    await vi.advanceTimersByTimeAsync(30_000)
    expect(client.getSubscription).toHaveBeenCalledTimes(reads)
  })

  it('gives up after 10 minutes and says the payment is not seen yet', async () => {
    const client = createClient([createResponse()])
    const { subscription } = mountSubscription(client)
    await subscription.load()
    await subscription.buy('plus')

    await vi.advanceTimersByTimeAsync(600_000)

    expect(subscription.waiting.value).toBe(false)
    expect(subscription.problem.value).toBe(NOT_SEEN)
    const reads = client.getSubscription.mock.calls.length
    await vi.advanceTimersByTimeAsync(30_000)
    expect(client.getSubscription).toHaveBeenCalledTimes(reads)
  })

  it('is still waiting just before the 10 minutes end', async () => {
    const { subscription } = mountSubscription(createClient([createResponse()]))
    await subscription.load()
    await subscription.buy('plus')

    await vi.advanceTimersByTimeAsync(599_000)

    expect(subscription.waiting.value).toBe(true)
    expect(subscription.problem.value).toBe('')
  })

  it('stops asking the server when the view is removed', async () => {
    const client = createClient([createResponse()])
    const { subscription, wrapper } = mountSubscription(client)
    await subscription.load()
    await subscription.buy('plus')
    await vi.advanceTimersByTimeAsync(3000)
    const reads = client.getSubscription.mock.calls.length

    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(60_000)

    expect(client.getSubscription).toHaveBeenCalledTimes(reads)
  })

  it('shows the problem and asks no more when the browser cannot open the page', async () => {
    const client = createClient([createResponse()])
    const openExternal = vi.fn<OpenExternal>(() => Promise.resolve(asFailure('not a web address')))
    const { subscription } = mountSubscription(client, openExternal)
    await subscription.load()

    await subscription.buy('plus')
    await vi.advanceTimersByTimeAsync(30_000)

    expect(subscription.problem.value).not.toBe('')
    expect(subscription.waiting.value).toBe(false)
    expect(client.getSubscription).toHaveBeenCalledTimes(1)
  })

  it('shows the problem and opens nothing when the server refuses the payment page', async () => {
    const client = createClient([createResponse()])
    client.createCheckout.mockRejectedValue(
      createFailure('failed_precondition', 'the sync server does not sell subscriptions'),
    )
    const { subscription, openExternal } = mountSubscription(client)
    await subscription.load()

    await subscription.buy('plus')

    expect(subscription.problem.value).toContain('the sync server does not sell subscriptions')
    expect(openExternal).not.toHaveBeenCalled()
  })

  it('reads the subscription again and opens nothing when the server says it is active already', async () => {
    const client = createClient([createResponse(), ACTIVE])
    client.createCheckout.mockRejectedValue(
      createFailure('already_exists', 'the account has an active subscription'),
    )
    const { subscription, openExternal } = mountSubscription(client)
    await subscription.load()

    await subscription.buy('plus')

    expect(subscription.phase.value).toBe('active')
    expect(subscription.canBuy.value).toBe(false)
    expect(openExternal).not.toHaveBeenCalled()
  })
})

describe('useSubscription plans', () => {
  it('offers to buy every plan with its name and size when the account has no subscription', async () => {
    const { subscription } = mountSubscription(createClient([createResponse()]))

    await subscription.load()

    expect(subscription.buyOffers.value).toStrictEqual([
      { id: 'basic', label: 'Basic, 10 GB' },
      { id: 'plus', label: 'Plus, 100 GB' },
      { id: 'pro', label: 'Pro, 250 GB' },
    ])
    expect(subscription.changeOffers.value).toStrictEqual([])
  })

  it('offers to buy every plan in each phase that is not active', async () => {
    for (const phase of ['grace', 'readOnly', 'deleted'] as const) {
      const { subscription } = mountSubscription(
        createClient([createResponse({ phase, plan: 'plus', quotaBytes: 100 * GIB })]),
      )

      await subscription.load()

      expect(
        subscription.buyOffers.value.map((one) => one.id),
        phase,
      ).toStrictEqual(['basic', 'plus', 'pro'])
      expect(subscription.changeOffers.value, phase).toStrictEqual([])
    }
  })

  it('offers to change to every other plan and to buy none while the subscription is active', async () => {
    const { subscription } = mountSubscription(createClient([ACTIVE_ON_PLUS]))

    await subscription.load()

    expect(subscription.buyOffers.value).toStrictEqual([])
    expect(subscription.changeOffers.value).toStrictEqual([
      { id: 'basic', label: 'Basic, 10 GB' },
      { id: 'pro', label: 'Pro, 250 GB' },
    ])
  })

  it('offers nothing when the device has no identity', async () => {
    const { subscription } = mountSubscription(
      createClient([createResponse({ hasIdentity: false, phase: '' })]),
    )

    await subscription.load()

    expect(subscription.buyOffers.value).toStrictEqual([])
    expect(subscription.changeOffers.value).toStrictEqual([])
  })

  it('names the plan with its size, also after the subscription ended', async () => {
    const ended = createResponse({ phase: 'readOnly', plan: 'plus', quotaBytes: 100 * GIB })
    const { subscription } = mountSubscription(createClient([ended]))

    await subscription.load()

    expect(subscription.planLine.value).toBe('Plus, 100 GB')
  })

  it('names only the size of a plan the client does not know', async () => {
    const unknown = createResponse({ ...ACTIVE_ON_PLUS, plan: 'gold', quotaBytes: 50 * GIB })
    const { subscription } = mountSubscription(createClient([unknown]))

    await subscription.load()

    expect(subscription.planLine.value).toBe('50 GB')
  })

  it('has no plan line without a subscription', async () => {
    const { subscription } = mountSubscription(createClient([createResponse()]))

    await subscription.load()

    expect(subscription.planLine.value).toBe('')
    expect(subscription.usage.value).toBe('')
  })

  it('shows the space used against the plan while the subscription is active', async () => {
    const { subscription } = mountSubscription(createClient([ACTIVE_ON_PLUS]))

    await subscription.load()

    expect(subscription.usage.value).toBe('2 GB of 100 GB used')
    expect(subscription.isFull.value).toBe(false)
  })

  it('shows the used space above the plan and says the cloud is full', async () => {
    const over = createResponse({
      ...ACTIVE_ON_PLUS,
      plan: 'basic',
      quotaBytes: 10 * GIB,
      usedBytes: 20 * GIB,
    })
    const { subscription } = mountSubscription(createClient([over]))

    await subscription.load()

    expect(subscription.usage.value).toBe('20 GB of 10 GB used')
    expect(subscription.isFull.value).toBe(true)
  })

  it('says the cloud is full when the used space equals the plan', async () => {
    const full = createResponse({
      ...ACTIVE_ON_PLUS,
      plan: 'basic',
      quotaBytes: 10 * GIB,
      usedBytes: 10 * GIB,
    })
    const { subscription } = mountSubscription(createClient([full]))

    await subscription.load()

    expect(subscription.isFull.value).toBe(true)
  })

  it('has no offers when the server cannot list the plans, and shows the problem', async () => {
    const client = createClient([createResponse()])
    client.listPlans.mockRejectedValue(
      createFailure('failed_precondition', 'the sync server does not sell subscriptions'),
    )
    const { subscription } = mountSubscription(client)

    await subscription.load()

    expect(subscription.buyOffers.value).toStrictEqual([])
    expect(subscription.problem.value).toContain('the sync server does not sell subscriptions')
  })
})

describe('useSubscription buy of a plan', () => {
  it('asks the payment page of the plan the person chose', async () => {
    const client = createClient([createResponse()])
    const { subscription } = mountSubscription(client)
    await subscription.load()

    await subscription.buy('pro')

    expect(client.createCheckout).toHaveBeenCalledExactlyOnceWith('pro')
  })
})

describe('useSubscription change of the plan', () => {
  it('asks the server to change the plan and shows the new plan at once', async () => {
    const client = createClient([ACTIVE_ON_PLUS])
    const { subscription, openExternal } = mountSubscription(client)
    await subscription.load()

    await subscription.change('pro')

    expect(client.changePlan).toHaveBeenCalledExactlyOnceWith('pro')
    expect(subscription.planLine.value).toBe('Pro, 250 GB')
    expect(subscription.changeOffers.value.map((one) => one.id)).toStrictEqual(['basic', 'plus'])
    expect(openExternal).not.toHaveBeenCalled()
    expect(subscription.waiting.value).toBe(false)
  })

  it('shows the problem and keeps the plan when the server refuses the change', async () => {
    const client = createClient([ACTIVE_ON_PLUS])
    client.changePlan.mockRejectedValue(
      createFailure('unavailable', 'Could not reach the sync server'),
    )
    const { subscription } = mountSubscription(client)
    await subscription.load()

    await subscription.change('pro')

    expect(subscription.problem.value).toContain('Could not reach the sync server')
    expect(subscription.planLine.value).toBe('Plus, 100 GB')
  })

  it('reads the subscription again when the server says the plan is the plan of the account', async () => {
    const client = createClient([
      ACTIVE_ON_PLUS,
      createResponse({ ...ACTIVE_ON_PLUS, plan: 'pro', quotaBytes: 250 * GIB }),
    ])
    client.changePlan.mockRejectedValue(
      createFailure('already_exists', 'the account has this plan already'),
    )
    const { subscription } = mountSubscription(client)
    await subscription.load()

    await subscription.change('pro')

    expect(subscription.planLine.value).toBe('Pro, 250 GB')
  })

  it('reads the subscription again when the server says there is no active subscription', async () => {
    const client = createClient([
      ACTIVE_ON_PLUS,
      createResponse({ phase: 'grace', plan: 'plus', quotaBytes: 100 * GIB }),
    ])
    client.changePlan.mockRejectedValue(
      createFailure('failed_precondition', 'the account has no active subscription to change'),
    )
    const { subscription } = mountSubscription(client)
    await subscription.load()

    await subscription.change('pro')

    expect(subscription.phase.value).toBe('grace')
    expect(subscription.changeOffers.value).toStrictEqual([])
  })
})
