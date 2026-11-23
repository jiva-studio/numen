<script setup lang="ts">
/**
 * The card of the subscription of the account on the page "Identity and licenses": the title, what the
 * server says of the subscription, and the one action, to buy it.
 */
import { onMounted } from 'vue'
import type { Result } from '@numen/plugin-sdk'
import { Button } from '@numen/ui'
import type { SubscriptionClient } from '../api/subscription'
import { useSubscription } from '../model/subscription'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  client?: SubscriptionClient | undefined
  openExternal: (url: string) => Promise<Result<null>>
  now?: (() => Date) | undefined
}>()

/* --------------------------------- State ---------------------------------- */
const { buyOffers, changeOffers, waiting, problem, view, load, buy, change } = useSubscription({
  openExternal: props.openExternal,
  client: props.client,
  now: props.now,
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
    <div v-if="buyOffers.length > 0 || changeOffers.length > 0" class="subscription__controls">
      <Button
        v-for="offer in buyOffers"
        :key="`buy-${offer.id}`"
        type="button"
        variant="solid"
        :disabled="waiting"
        @click="onBuy(offer.id)"
      >
        {{ waiting ? words.waitingForPayment : `${words.buy} ${offer.label}` }}
      </Button>
      <Button
        v-for="offer in changeOffers"
        :key="`change-${offer.id}`"
        type="button"
        variant="outline"
        @click="onChange(offer.id)"
      >
        {{ `Change plan to ${offer.name}` }}
      </Button>
    </div>
    <p v-if="problem !== ''" role="alert" class="subscription__problem">{{ problem }}</p>
  </section>
</template>
