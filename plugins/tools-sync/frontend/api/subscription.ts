/**
 * The JSON messages of the subscription methods the plugin's backend serves, and the client that calls
 * them. Keys and meaning match the Go messages in backend/api; testdata/api holds the fixtures both sides
 * are tested against.
 */
import { ReportedError, type Api, type ApiError } from '@numen/plugin-sdk'

import { REQUIREMENT_UNMET } from './call'

import type { SubscriptionPhase } from '../lib/subscriptionView'

/**
 * The phase is empty when the device has no identity. A time is RFC 3339, and empty when the server sent none.
 * The plan is the id of the plan, empty while the phase is none. The used bytes may be above the quota.
 * isTrial is true while a trial runs that no payment has followed, and the backend leaves it out otherwise.
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
  readonly isTrial?: boolean | undefined
}

/** A plan the server sells. The price is for one period in cents of the currency, tax included, and is information only. */
export interface PlanResponse {
  readonly id: string
  readonly name: string
  readonly quotaBytes: number
  readonly priceCents: number
  readonly currency: string
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

interface CreatePortalSessionResponse {
  readonly portalUrl: string
}

export interface SubscriptionClient {
  getSubscription(): Promise<SubscriptionResponse>
  /** The plans ascending by quota. */
  listPlans(): Promise<readonly PlanResponse[]>
  /** The address of the payment page for the plan with this id. */
  createCheckout(plan: string): Promise<string>
  /** The subscription after the active subscription moved to the plan with this id. */
  changePlan(plan: string): Promise<SubscriptionResponse>
  /** The address of the page of the payment provider where the person manages the subscription. */
  createPortalSession(): Promise<string>
}

/** A failed call: the message of the backend and its code, such as `unavailable` or `already_exists`. */
interface SubscriptionFailure extends Error {
  readonly code: string
}

const NO_IDENTITY: SubscriptionResponse = {
  hasIdentity: false,
  phase: '',
  periodEndAt: '',
  writesEndAt: '',
  deleteAt: '',
  plan: '',
  quotaBytes: 0,
  usedBytes: 0,
}

const createFailure = (error: ApiError): SubscriptionFailure =>
  Object.assign(new ReportedError(error.message), { code: error.code })

export function createSubscriptionClient(api: Api): SubscriptionClient {
  async function call<Req extends object, Res>(
    method: string,
    request: Req,
    ifNoIdentity?: Res,
  ): Promise<Res> {
    const result = await api.call<Req, Res>(method, request)
    if (!result.ok && ifNoIdentity !== undefined && result.error.code === REQUIREMENT_UNMET) {
      return ifNoIdentity
    }
    if (!result.ok) throw createFailure(result.error)
    return result.value
  }

  return {
    getSubscription: () =>
      call<Record<string, never>, SubscriptionResponse>('GetSubscription', {}, NO_IDENTITY),
    listPlans: async () =>
      (await call<Record<string, never>, ListPlansResponse>('ListPlans', {})).plans,
    createCheckout: async (plan) =>
      (await call<{ plan: string }, CreateCheckoutResponse>('CreateCheckout', { plan }))
        .checkoutUrl,
    changePlan: async (plan) =>
      (await call<{ plan: string }, ChangePlanResponse>('ChangePlan', { plan })).subscription,
    createPortalSession: async () =>
      (await call<Record<string, never>, CreatePortalSessionResponse>('CreatePortalSession', {}))
        .portalUrl,
  }
}

export const getFailureCode = (failure: unknown): string =>
  failure instanceof Error && 'code' in failure ? String(failure.code) : ''
