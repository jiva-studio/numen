<script setup lang="ts">
/**
 * The card of the subscription of the account on the page "Identity and licenses": the title with the plan
 * as its badge, what the server says of the subscription, the plan with the space used and the end of the
 * paid period, the plans to choose from when there is no subscription, and to change or manage it when there is.
 */
import { onMounted } from 'vue'
import type { PluginSettings, Result } from '@numen/plugin-sdk'
import { Button, Select } from '@numen/ui'
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
const { problem, status, load, change, manage } = session
const { waiting, buy } = payment
const { buyOffers, changeOffers, canManage } = offers
const { view, planLine, usage, usedPercent, isFull, validity } = summary

const toChoice = (offer: PlanOffer) => ({
  id: offer.id,
  text: offer.name,
  detail: [offer.size, offer.price].filter((part) => part !== '').join(', '),
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
  <section class="settings-card subscription" :aria-label="words.subscriptionTitle">
    <div class="subscription__header">
      <h3 class="subscription__title">{{ words.subscriptionTitle }}</h3>
      <span
        v-if="view.badge !== ''"
        class="subscription__badge"
        :class="`subscription__badge--${view.tone}`"
      >
        {{ view.badge }}
      </span>
    </div>
    <p v-if="view.text !== ''" class="subscription__text">{{ view.text }}</p>
    <dl
      v-if="planLine !== '' || usage !== '' || validity !== undefined"
      class="subscription__facts"
    >
      <template v-if="planLine !== ''">
        <dt>{{ words.planRow }}</dt>
        <dd>{{ planLine }}</dd>
      </template>
      <template v-if="usage !== ''">
        <dt>{{ words.storageRow }}</dt>
        <dd class="subscription__storage">
          <div
            role="progressbar"
            class="subscription__meter"
            :aria-label="words.usagePercentName"
            aria-valuemin="0"
            aria-valuemax="100"
            :aria-valuenow="usedPercent"
          >
            <div
              class="subscription__meter-fill"
              :class="{ 'subscription__meter-fill--full': isFull }"
              :style="{ inlineSize: `${usedPercent}%` }"
            />
          </div>
          <span class="subscription__usage">{{ usage }}</span>
        </dd>
      </template>
      <template v-if="validity !== undefined">
        <dt>{{ validity.label }}</dt>
        <dd>{{ validity.date }}</dd>
      </template>
    </dl>
    <p v-if="isFull" class="subscription__full">{{ words.cloudFull }}</p>
    <div
      v-if="status === 'unreachable' && openSettings !== undefined"
      class="subscription__actions"
    >
      <Button type="button" variant="outline" @click="props.openSettings?.()">
        {{ words.openSettings }}
      </Button>
    </div>
    <p v-if="waiting" role="status" class="subscription__waiting">
      <strong>{{ words.waitingForPayment }}</strong>
      {{ words.waitingForPaymentText }}
    </p>
    <template v-else-if="buyOffers.length > 0">
      <p class="subscription__choose">{{ words.choosePlan }}</p>
      <ul class="subscription__plans">
        <li v-for="offer in buyOffers" :key="offer.id" class="subscription__tile">
          <span class="subscription__tile-name">{{ offer.name }}</span>
          <span class="subscription__tile-size">{{ offer.size }}</span>
          <span v-if="offer.price !== ''" class="subscription__tile-price">{{ offer.price }}</span>
          <Button
            type="button"
            variant="solid"
            :aria-label="words.chooseNamed(offer.name)"
            @click="onBuy(offer.id)"
          >
            {{ words.choose }}
          </Button>
        </li>
      </ul>
    </template>
    <div v-if="changeOffers.length > 0 || canManage" class="subscription__actions">
      <Select
        v-if="changeOffers.length > 0"
        class="subscription__change"
        model-value=""
        :choices="changeOffers.map(toChoice)"
        :placeholder="words.changePlan"
        :name="words.changePlanChoices"
        :aria-label="words.changePlan"
        @update:model-value="onChange"
      />
      <Button v-if="canManage" type="button" variant="outline" @click="onManage">
        {{ words.manageSubscription }}
      </Button>
    </div>
    <p v-if="problem !== ''" role="alert" class="subscription__problem">{{ problem }}</p>
  </section>
</template>
