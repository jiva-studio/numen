<script setup lang="ts">
import { formatErrorMessage } from '@numen/plugin-sdk'
/**
 * The card of the subscription of the identity on the page "Identity and licenses", an island card with
 * a row per plan that buys it on a click when there is no subscription, and the
 * rows of the plan, the end of the paid period and the space used with the actions to change or manage
 * it in the menu "..." when there is. A plan chosen in "Change plan" is changed to only after the person confirms
 * the switch. While a payment is awaited, a layer over the rows keeps them in place, says so and offers to
 * stop waiting.
 */
import { computed, onMounted, ref } from 'vue'
import { ChevronRight, RotateCcw, ServerOff, Settings } from '@lucide/vue'
import { Button, CardAction, IslandCard, Spinner } from '@numen/ui'
import { formatDate, getOfferDetail, getOfferLine, type PlanOffer } from '../lib/subscriptionView'
import { useSubscription } from '../model/subscription'
import { WORDS as words } from '../words'
import AccountMenu from './AccountMenu.vue'
import DeleteCloudDataDialog from './DeleteCloudDataDialog.vue'
import SwitchPlanDialog from './SwitchPlanDialog.vue'
import type { SubscriptionCardProps } from './SubscriptionCard.types'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<SubscriptionCardProps>()

/* --------------------------------- State ---------------------------------- */
const { session, payment, offers, summary, accountData, start } = useSubscription({
  openExternal: props.openExternal,
  client: props.client,
  accountClient: props.accountClient,
  saveFile: props.saveFile,
  now: props.now,
  settings: props.settings,
  refreshMs: props.refreshMs,
})
const { problem, plansProblem } = session
const { change, manage } = session.actions
const { status } = session.account
const { waiting, starting, startPurchase, cancel } = payment
const { buyOffers, changeOffers, currentOffer, canManage } = offers
const { view, planLine, usage, usedPercent, isFull, validity } = summary
const { phase } = session.account

const hasSettingsAction = computed(
  () => status.value === 'unreachable' && props.openSettings !== undefined,
)
const isPaying = computed(() => waiting.value || starting.value)
const isDeleting = computed(() => accountData?.state.isDeleting.value === true)
const canChangeAccount = computed(
  () => accountData !== undefined && status.value === 'ready' && !isDeleting.value,
)
const deletingText = computed(() =>
  words.deletingText(formatDate(accountData?.state.deleteAt.value ?? '')),
)
const isConfirmingDelete = ref(false)

/** The plan chosen in the menu "Change plan" that waits for the person to confirm the switch. */
const pendingOffer = ref<PlanOffer | undefined>(undefined)
const currentLine = computed(() =>
  currentOffer.value === undefined ? planLine.value : getOfferLine(currentOffer.value),
)

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  start().catch(onFailure)
})

/* -------------------------------- Handlers -------------------------------- */
function onBuy(plan: string): void {
  startPurchase(plan).catch(onFailure)
}

function onChoosePlan(offer: PlanOffer): void {
  pendingOffer.value = offer
}

function onConfirmSwitch(): void {
  const offer = pendingOffer.value
  pendingOffer.value = undefined
  if (offer !== undefined) change(offer.id).catch(onFailure)
}

function onCancelSwitch(): void {
  pendingOffer.value = undefined
}

function onManage(): void {
  manage().catch(onFailure)
}

function onAskDelete(): void {
  isConfirmingDelete.value = true
}

function onCancelDelete(): void {
  isConfirmingDelete.value = false
}

function onConfirmDelete(): void {
  isConfirmingDelete.value = false
  accountData?.actions.remove().catch(onFailure)
}

function onRestore(): void {
  accountData?.actions.restore().catch(onFailure)
}

function onExport(): void {
  accountData?.actions.exportData().catch(onFailure)
}

function onFailure(failure: unknown): void {
  problem.value = failure instanceof Error ? failure.message : formatErrorMessage(failure)
}
</script>

<template>
  <IslandCard
    class="subscription"
    :title="words.subscriptionTitle"
    :aria-label="words.subscriptionTitle"
  >
    <template #aside>
      <span v-if="isDeleting" class="subscription__badge subscription__badge--warning">
        {{ words.deletingBadge }}
      </span>
      <span
        v-else-if="view.badge !== ''"
        class="subscription__badge"
        :class="`subscription__badge--${view.tone}`"
      >
        {{ view.badge }}
      </span>
      <CardAction
        v-if="isDeleting"
        is-primary
        :label="words.restoreAccount"
        :is-disabled="accountData?.state.isBusy.value ?? false"
        @click="onRestore"
      >
        <RotateCcw :size="14" aria-hidden="true" />
      </CardAction>
      <template v-else>
        <CardAction
          v-if="hasSettingsAction"
          is-primary
          :label="words.openSettings"
          @click="props.openSettings?.()"
        >
          <Settings :size="14" aria-hidden="true" />
        </CardAction>
        <AccountMenu
          v-if="canChangeAccount || canManage || changeOffers.length > 0"
          :has-account-actions="canChangeAccount"
          :can-manage="canManage"
          :offers="changeOffers"
          :current-id="currentOffer?.id ?? ''"
          :is-exporting="accountData?.state.isExporting.value === true"
          @choose-plan="onChoosePlan"
          @manage="onManage"
          @export="onExport"
          @delete="onAskDelete"
        />
      </template>
    </template>
    <div v-if="isDeleting" class="subscription__content">
      <p class="subscription__text">{{ deletingText }}</p>
    </div>
    <div v-else class="subscription__content" :inert="isPaying || undefined" :aria-busy="isPaying">
      <p v-if="isFull" class="subscription__full">{{ words.cloudFull }}</p>
      <p v-if="status === 'unreachable'" class="subscription__text subscription__away">
        <ServerOff :size="16" aria-hidden="true" />
        <span>{{ view.text }}</span>
      </p>
      <p v-else-if="view.text !== ''" class="subscription__text">{{ view.text }}</p>
      <div v-if="planLine !== ''" class="subscription__row">
        <div class="subscription__label">
          <span class="subscription__name">{{ words.planRow }}</span>
          <span class="subscription__detail">{{ planLine }}</span>
        </div>
      </div>
      <div v-if="validity !== undefined" class="subscription__row">
        <div class="subscription__label">
          <span class="subscription__name">{{ validity.label }}</span>
          <span class="subscription__detail">{{ validity.date }}</span>
        </div>
      </div>
      <div v-if="usage !== ''" class="subscription__row">
        <div class="subscription__label">
          <span class="subscription__name">{{ words.storageRow }}</span>
          <span v-if="!isFull" class="subscription__detail">{{ words.storageDetail }}</span>
        </div>
        <div class="subscription__value">
          <div class="subscription__meter" :class="{ 'subscription__meter--full': isFull }">
            <div
              role="progressbar"
              class="subscription__meter-bar"
              :aria-label="words.usagePercentName"
              aria-valuemin="0"
              aria-valuemax="100"
              :aria-valuenow="usedPercent"
            >
              <div class="subscription__meter-fill" :style="{ inlineSize: `${usedPercent}%` }" />
            </div>
            <span class="subscription__usage">{{ usage }}</span>
          </div>
        </div>
      </div>
      <button
        v-for="offer in buyOffers"
        :key="offer.id"
        type="button"
        class="subscription__row subscription__offer"
        :aria-label="words.chooseNamed(offer.name, getOfferDetail(offer))"
        @click="onBuy(offer.id)"
      >
        <span class="subscription__label">
          <span class="subscription__name">{{ offer.name }}</span>
          <span class="subscription__detail">{{ words.syncedFiles(offer.size) }}</span>
        </span>
        <span class="subscription__value">
          <span v-if="offer.price !== ''" class="subscription__price">
            {{ offer.price }}
            <small>{{ words.perMonth }}</small>
          </span>
          <ChevronRight class="subscription__chevron" :size="16" aria-hidden="true" />
        </span>
      </button>
    </div>
    <div v-if="isPaying" class="subscription__waiting">
      <p role="status" class="subscription__waiting-text">
        <Spinner class="subscription__spinner" />
        <span>{{ words.waitingForPayment }}</span>
      </p>
      <Button type="button" variant="ghost" size="small" @click="cancel">
        {{ words.stopWaiting }}
      </Button>
    </div>
    <p v-if="plansProblem !== '' && status === 'ready'" role="alert" class="subscription__problem">
      {{ plansProblem }}
    </p>
    <p v-if="problem !== '' && status !== 'unreachable'" role="alert" class="subscription__problem">
      {{ problem }}
    </p>
    <p
      v-if="(accountData?.state.problem.value ?? '') !== ''"
      role="alert"
      class="subscription__problem"
    >
      {{ accountData?.state.problem.value }}
    </p>
    <DeleteCloudDataDialog
      v-if="isConfirmingDelete"
      :has-subscription="phase === 'active'"
      @confirm="onConfirmDelete"
      @cancel="onCancelDelete"
    />
    <SwitchPlanDialog
      v-if="pendingOffer"
      :offer="pendingOffer"
      :current-line="currentLine"
      :renewal="validity?.date ?? ''"
      @confirm="onConfirmSwitch"
      @cancel="onCancelSwitch"
    />
  </IslandCard>
</template>
