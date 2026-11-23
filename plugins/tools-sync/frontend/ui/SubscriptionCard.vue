<script setup lang="ts">
/**
 * The card of the subscription of the account on the page "Identity and licenses", an island card with
 * the plan as its badge: a row per plan that buys it on a click when there is no subscription, and the
 * rows of the plan, the end of the paid period and the space used with the actions to change or manage
 * it when there is. While a payment is awaited, a layer over the rows keeps them in place, says so and
 * offers to stop waiting.
 */
import { computed, onMounted } from 'vue'
import { ChevronRight } from '@lucide/vue'
import type { PluginSettings, Result } from '@numen/plugin-sdk'
import { Button, IslandCard, Select, Spinner } from '@numen/ui'
import type { SubscriptionClient } from '../api/subscription'
import type { PlanOffer } from '../lib/subscriptionView'
import { useSubscription, type UseSubscriptionOptions } from '../model/subscription'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  client?: SubscriptionClient | undefined
  openExternal: (url: string) => Promise<Result<null>>
  now?: (() => Date) | undefined
  settings?: Pick<PluginSettings, 'get' | 'onChange'> | undefined
  openSettings?: (() => void) | undefined
  refreshMs?: UseSubscriptionOptions['refreshMs']
}>()

/* --------------------------------- State ---------------------------------- */
const { session, payment, offers, summary } = useSubscription({
  openExternal: props.openExternal,
  client: props.client,
  now: props.now,
  settings: props.settings,
  refreshMs: props.refreshMs,
})
const { problem, load, change, manage } = session
const { status } = session.account
const { waiting, starting, buy, cancel } = payment
const { buyOffers, changeOffers, canManage } = offers
const { view, planLine, usage, usedPercent, isFull, validity } = summary

const hasSettingsAction = computed(
  () => status.value === 'unreachable' && props.openSettings !== undefined,
)
const hasManageAction = computed(() => canManage.value && validity.value === undefined)
const isPaying = computed(() => waiting.value || starting.value)

const getOfferDetail = (offer: PlanOffer): string =>
  [offer.size, offer.price === '' ? '' : words.pricePerMonth(offer.price)]
    .filter((part) => part !== '')
    .join(', ')

const toChoice = (offer: PlanOffer) => ({
  id: offer.id,
  text: offer.name,
  detail: getOfferDetail(offer),
})

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  load().catch(onFailure)
})

/* -------------------------------- Handlers -------------------------------- */
function onBuy(plan: string): void {
  buy(plan).catch(onFailure)
}

function onChange(plan: string): void {
  change(plan).catch(onFailure)
}

function onManage(): void {
  manage().catch(onFailure)
}

function onFailure(failure: unknown): void {
  problem.value = failure instanceof Error ? failure.message : String(failure)
}
</script>

<template>
  <IslandCard
    class="subscription"
    :title="words.subscriptionTitle"
    :aria-label="words.subscriptionTitle"
  >
    <template v-if="view.badge !== ''" #aside>
      <span class="subscription__badge" :class="`subscription__badge--${view.tone}`">
        {{ view.badge }}
      </span>
    </template>
    <div class="subscription__content" :inert="isPaying || undefined" :aria-busy="isPaying">
      <p v-if="isFull" class="subscription__full">{{ words.cloudFull }}</p>
      <p v-if="view.text !== ''" class="subscription__text">{{ view.text }}</p>
      <div v-if="planLine !== ''" class="subscription__row">
        <div class="subscription__label">
          <span class="subscription__name">{{ words.planRow }}</span>
          <span class="subscription__detail">{{ planLine }}</span>
        </div>
        <div v-if="changeOffers.length > 0" class="subscription__value">
          <Select
            class="subscription__change"
            :class="{ 'subscription__change--solid': isFull }"
            model-value=""
            :choices="changeOffers.map(toChoice)"
            :placeholder="words.changePlan"
            :name="words.changePlanChoices"
            :aria-label="words.changePlan"
            @update:model-value="onChange"
          />
        </div>
      </div>
      <div v-if="validity !== undefined" class="subscription__row">
        <div class="subscription__label">
          <span class="subscription__name">{{ validity.label }}</span>
          <span class="subscription__detail">{{ validity.date }}</span>
        </div>
        <div v-if="canManage" class="subscription__value">
          <Button type="button" variant="outline" @click="onManage">
            {{ words.manageSubscription }}
          </Button>
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
      <div v-if="hasSettingsAction || hasManageAction" class="subscription__actions">
        <Button
          v-if="hasSettingsAction"
          type="button"
          variant="outline"
          @click="props.openSettings?.()"
        >
          {{ words.openSettings }}
        </Button>
        <Button v-if="hasManageAction" type="button" variant="outline" @click="onManage">
          {{ words.manageSubscription }}
        </Button>
      </div>
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
    <p v-if="problem !== ''" role="alert" class="subscription__problem">{{ problem }}</p>
  </IslandCard>
</template>
