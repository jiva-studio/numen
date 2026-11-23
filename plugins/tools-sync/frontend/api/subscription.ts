/**
 * The JSON messages of the subscription methods the plugin's backend serves, and the client that calls
 * them. Keys and meaning match the Go messages in backend/api; testdata/api holds the fixtures both sides
 * are tested against.
 */
import type { Api, ApiError } from '@numen/plugin-sdk'

import type { SubscriptionPhase } from '../lib/subscriptionView'

/** The phase is empty when the device has no identity. A time is RFC 3339, and empty when the server sent none. */
interface SubscriptionResponse {
  readonly hasIdentity: boolean
  readonly phase: SubscriptionPhase | ''
  readonly periodEndAt: string
  readonly writesEndAt: string
  readonly deleteAt: string
}

interface CreateCheckoutResponse {
  readonly checkoutUrl: string
}

export interface SubscriptionClient {
  getSubscription(): Promise<SubscriptionResponse>
  /** The address of the payment page. */
  createCheckout(): Promise<string>
}

/** A failed call: the message of the backend and its code, such as `unavailable` or `already_exists`. */
interface SubscriptionFailure extends Error {
  readonly code: string
}

type Empty = Record<string, never>

const createFailure = (error: ApiError): SubscriptionFailure =>
  Object.assign(new Error(error.message), { code: error.code })

export function createSubscriptionClient(api: Api): SubscriptionClient {
  async function call<Res>(method: string): Promise<Res> {
    const result = await api.call<Empty, Res>(method, {})
    if (!result.ok) throw createFailure(result.error)
    return result.value
  }

  return {
    getSubscription: () => call<SubscriptionResponse>('GetSubscription'),
    createCheckout: async () => (await call<CreateCheckoutResponse>('CreateCheckout')).checkoutUrl,
  }
}

export const getFailureCode = (failure: unknown): string =>
  failure instanceof Error && 'code' in failure ? String(failure.code) : ''
