<script setup lang="ts">
/**
 * The card of the subscription of the account on the page "Identity and licenses": the title, what the
 * server says of the subscription, the plan with the space used, and the actions: to buy a plan, to change it and to manage the subscription.
 */
import { onMounted } from 'vue'
import type { Result } from '@numen/plugin-sdk'
import { Button } from '@numen/ui'
import type { SubscriptionClient } from '../api/subscription'
import type { ProductOffer } from '../lib/subscriptionView'
import { useSubscription } from '../model/subscription'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  client?: SubscriptionClient | undefined
  openExternal: (url: string) => Promise<Result<null>>
  now?: (() => Date) | undefined
}>()

/* --------------------------------- State ---------------------------------- */
const {
  buyOffers,
  changeOffers,
  productOffers,
  ownedProducts,
  canManage,
  planLine,
  usage,
  isFull,
  waiting,
  problem,
  view,
  load,
  buy,
  buyProduct,
  change,
  manage,
} = useSubscription({
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

function onBuyProduct(offer: ProductOffer): void {
  buyProduct(offer).catch(onFailure)
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
    <p v-if="planLine !== ''" class="subscription__plan">{{ planLine }}</p>
    <p v-if="usage !== ''" class="subscription__usage">{{ usage }}</p>
    <p v-if="isFull" class="subscription__full">{{ words.cloudFull }}</p>
    <ul v-if="ownedProducts.length > 0" class="subscription__products" :aria-label="words.productsTitle">
      <li v-for="name in ownedProducts" :key="name" class="subscription__product">
        <span>{{ name }}</span>
        <span class="subscription__badge subscription__badge--positive">{{ words.owned }}</span>
      </li>
    </ul>
    <div
      v-if="buyOffers.length > 0 || changeOffers.length > 0 || productOffers.length > 0 || canManage"
      class="subscription__controls"
    >
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
        v-for="offer in productOffers"
        :key="`buy-product-${offer.id}`"
        type="button"
        variant="solid"
        :disabled="waiting"
        @click="onBuyProduct(offer)"
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
        {{ words.changePlanTo(offer.name) }}
      </Button>
      <Button v-if="canManage" type="button" variant="outline" @click="onManage">
        {{ words.manageSubscription }}
      </Button>
    </div>
    <p v-if="problem !== ''" role="alert" class="subscription__problem">{{ problem }}</p>
  </section>
</template>
