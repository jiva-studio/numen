/**
 * The subscription methods of the service as the settings view calls them: the fixtures of testdata/api are
 * the JSON the Go messages encode to, and a failed call throws an error with the code of the backend.
 */
import { describe, expect, it } from 'vitest'
import { createApi } from '@numen/plugin-sdk'
import { SERVICE } from './vaults'
import { createSubscriptionClient } from './subscription'
import type { SubscriptionClient } from './subscription'

type SubscriptionResponse = Awaited<ReturnType<SubscriptionClient['getSubscription']>>
interface CreateCheckoutResponse {
  readonly checkoutUrl: string
}

const fixtures = import.meta.glob('../../testdata/api/*.json', {
  eager: true,
  import: 'default',
}) as Record<string, unknown>

const read = (name: string): unknown => fixtures[`../../testdata/api/${name}.json`]

const ORIGIN = 'http://window'

interface Sent {
  readonly url: string
  readonly body: unknown
}

const createClient = (respond: () => Response) => {
  const sent: Sent[] = []
  const client = createSubscriptionClient(
    createApi(SERVICE, {
      origin: ORIGIN,
      fetch: async (url, init) => {
        sent.push({ url: String(url), body: JSON.parse(String(init?.body)) })
        return respond()
      },
    }),
  )
  return { client, sent }
}

describe('the fixtures of the subscription methods', () => {
  it('reads the GetSubscription fixture as SubscriptionResponse', () => {
    const response: SubscriptionResponse = {
      hasIdentity: true,
      phase: 'grace',
      periodEndAt: '2026-11-20T12:00:00Z',
      writesEndAt: '2026-11-27T12:00:00Z',
      deleteAt: '2027-01-26T12:00:00Z',
      plan: 'plus',
      quotaBytes: 107374182400,
      usedBytes: 2147483648,
    }

    expect(read('GetSubscription.request')).toStrictEqual({})
    expect(read('GetSubscription.response')).toStrictEqual(response)
  })

  it('reads the CreateCheckout fixture as CreateCheckoutResponse', () => {
    const response: CreateCheckoutResponse = { checkoutUrl: 'https://pay.example/checkout/abc' }

    expect(read('CreateCheckout.request')).toStrictEqual({ plan: 'plus' })
    expect(read('CreateCheckout.response')).toStrictEqual(response)
  })

  it('reads the ListPlans fixture as the plans ascending by quota', () => {
    expect(read('ListPlans.request')).toStrictEqual({})
    expect(read('ListPlans.response')).toStrictEqual({
      plans: [
        { id: 'basic', quotaBytes: 10737418240 },
        { id: 'plus', quotaBytes: 107374182400 },
        { id: 'pro', quotaBytes: 268435456000 },
      ],
    })
  })

  it('reads the ChangePlan fixture as the subscription after the change', () => {
    expect(read('ChangePlan.request')).toStrictEqual({ plan: 'pro' })
    expect(read('ChangePlan.response')).toStrictEqual({
      subscription: {
        hasIdentity: true,
        phase: 'active',
        periodEndAt: '2026-12-20T12:00:00Z',
        writesEndAt: '2026-12-27T12:00:00Z',
        deleteAt: '2027-02-18T12:00:00Z',
        plan: 'pro',
        quotaBytes: 268435456000,
        usedBytes: 1073741824,
      },
    })
  })
})

describe('the subscription client', () => {
  it('asks GetSubscription and answers its response', async () => {
    const answer = read('GetSubscription.response') as SubscriptionResponse
    const { client, sent } = createClient(() => Response.json(answer))

    const response = await client.getSubscription()

    expect(sent).toStrictEqual([{ url: `${ORIGIN}/${SERVICE}/GetSubscription`, body: {} }])
    expect(response).toStrictEqual(answer)
  })

  it('asks CreateCheckout and answers the address of the payment page', async () => {
    const answer = read('CreateCheckout.response') as CreateCheckoutResponse
    const { client, sent } = createClient(() => Response.json(answer))

    const url = await client.createCheckout('plus')

    expect(sent).toStrictEqual([
      { url: `${ORIGIN}/${SERVICE}/CreateCheckout`, body: { plan: 'plus' } },
    ])
    expect(url).toBe('https://pay.example/checkout/abc')
  })

  it('asks ListPlans and answers the plans', async () => {
    const answer = read('ListPlans.response') as { plans: unknown[] }
    const { client, sent } = createClient(() => Response.json(answer))

    const plans = await client.listPlans()

    expect(sent).toStrictEqual([{ url: `${ORIGIN}/${SERVICE}/ListPlans`, body: {} }])
    expect(plans).toStrictEqual(answer.plans)
  })

  it('asks ChangePlan with the plan and answers the subscription after the change', async () => {
    const answer = read('ChangePlan.response') as { subscription: SubscriptionResponse }
    const { client, sent } = createClient(() => Response.json(answer))

    const subscription = await client.changePlan('pro')

    expect(sent).toStrictEqual([{ url: `${ORIGIN}/${SERVICE}/ChangePlan`, body: { plan: 'pro' } }])
    expect(subscription).toStrictEqual(answer.subscription)
  })

  it('throws the message and the code of a failed call', async () => {
    const { client } = createClient(() =>
      Response.json(
        { code: 'already_exists', message: 'the account has an active subscription' },
        { status: 409 },
      ),
    )

    const answer = client.createCheckout('plus')

    await expect(answer).rejects.toMatchObject({
      message: 'the account has an active subscription',
      code: 'already_exists',
    })
  })
})
