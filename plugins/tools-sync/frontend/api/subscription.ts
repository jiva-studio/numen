/**
 * The JSON messages of the subscription methods the plugin's backend serves, and the client that calls
 * them. Keys and meaning match the Go messages in backend/api; testdata/api holds the fixtures both sides
 * are tested against.
 */
import type { Api, ApiError } from '@numen/plugin-sdk'

import type { SubscriptionPhase } from '../lib/subscriptionView'

/**
 * The phase is empty when the device has no identity. A time is RFC 3339, and empty when the server sent none.
 * The plan is the id of the plan, empty while the phase is none. The used bytes may be above the quota.
 */
export interface SubscriptionResponse {
  readonly hasIdentity: boolean
  readonly phase: SubscriptionPhase | ''
  readonly periodEndAt: string
  readonly writesEndAt: string
  readonly deleteAt: string
  readonly plan: string
  readonly quotaBytes: number
  readonly usedBytes: number
}

/** A plan the server sells. */
export interface PlanResponse {
  readonly id: string
  readonly quotaBytes: number
}

interface ListPlansResponse {
  readonly plans: readonly PlanResponse[]
}

interface CreateCheckoutResponse {
  readonly checkoutUrl: string
}

interface ChangePlanResponse {
  readonly subscription: SubscriptionResponse
}

export interface SubscriptionClient {
  getSubscription(): Promise<SubscriptionResponse>
  /** The plans ascending by quota. */
  listPlans(): Promise<readonly PlanResponse[]>
  /** The address of the payment page for the plan with this id. */
  createCheckout(plan: string): Promise<string>
  /** The subscription after the active subscription moved to the plan with this id. */
  changePlan(plan: string): Promise<SubscriptionResponse>
}

/** A failed call: the message of the backend and its code, such as `unavailable` or `already_exists`. */
interface SubscriptionFailure extends Error {
  readonly code: string
}

const createFailure = (error: ApiError): SubscriptionFailure =>
  Object.assign(new Error(error.message), { code: error.code })

export function createSubscriptionClient(api: Api): SubscriptionClient {
  async function call<Req extends object, Res>(method: string, request: Req): Promise<Res> {
    const result = await api.call<Req, Res>(method, request)
    if (!result.ok) throw createFailure(result.error)
    return result.value
  }

  return {
    getSubscription: () => call<Record<string, never>, SubscriptionResponse>('GetSubscription', {}),
    listPlans: async () =>
      (await call<Record<string, never>, ListPlansResponse>('ListPlans', {})).plans,
    createCheckout: async (plan) =>
      (await call<{ plan: string }, CreateCheckoutResponse>('CreateCheckout', { plan }))
        .checkoutUrl,
    changePlan: async (plan) =>
      (await call<{ plan: string }, ChangePlanResponse>('ChangePlan', { plan })).subscription,
  }
}

export const getFailureCode = (failure: unknown): string =>
  failure instanceof Error && 'code' in failure ? String(failure.code) : ''
