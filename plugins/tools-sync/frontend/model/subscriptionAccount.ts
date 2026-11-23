import { ref, type Ref } from 'vue'
import type { PlanResponse, SubscriptionResponse } from '../api/subscription'
import type {
  SubscriptionDates,
  SubscriptionPhase,
  SubscriptionStatus,
} from '../lib/subscriptionView'

const NO_DATES: SubscriptionDates = { periodEnd: '', writesEnd: '', deleteAt: '' }

/** What the card knows of the account: the last answer of the server and the plans it sells. */
export interface SubscriptionAccount {
  readonly status: Ref<SubscriptionStatus>
  readonly phase: Ref<SubscriptionPhase | ''>
  readonly dates: Ref<SubscriptionDates>
  readonly plan: Ref<string>
  readonly quotaBytes: Ref<number>
  readonly usedBytes: Ref<number>
  readonly isTrial: Ref<boolean>
  readonly plans: Ref<readonly PlanResponse[]>
}

export const getMessage = (failure: unknown): string =>
  failure instanceof Error ? failure.message : String(failure)

export function createAccount(): SubscriptionAccount {
  return {
    status: ref<SubscriptionStatus>('loading'),
    phase: ref<SubscriptionPhase | ''>(''),
    dates: ref<SubscriptionDates>(NO_DATES),
    plan: ref(''),
    quotaBytes: ref(0),
    usedBytes: ref(0),
    isTrial: ref(false),
    plans: ref<readonly PlanResponse[]>([]),
  }
}

/** Takes the answer of the server: an account without an identity has no phase, plan or space. */
export function applyAnswer(account: SubscriptionAccount, answer: SubscriptionResponse): void {
  account.status.value = answer.hasIdentity ? 'ready' : 'noIdentity'
  account.phase.value = answer.hasIdentity ? answer.phase : ''
  account.dates.value = answer.hasIdentity
    ? { periodEnd: answer.periodEndAt, writesEnd: answer.writesEndAt, deleteAt: answer.deleteAt }
    : NO_DATES
  account.plan.value = answer.hasIdentity ? answer.plan : ''
  account.quotaBytes.value = answer.hasIdentity ? answer.quotaBytes : 0
  account.usedBytes.value = answer.hasIdentity ? answer.usedBytes : 0
  account.isTrial.value = answer.hasIdentity && answer.isTrial === true
}
