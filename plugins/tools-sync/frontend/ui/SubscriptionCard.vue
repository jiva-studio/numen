<script setup lang="ts">
/**
 * The card of the subscription of the account on the page "Identity and licenses": the title, what the
 * server says of the subscription, the plan with the space used, and the actions: to buy a plan, to change it and to manage the subscription.
 */
import { onMounted } from 'vue'
import type { PluginSettings, Result } from '@numen/plugin-sdk'
import { Button } from '@numen/ui'
import type { SubscriptionClient } from '../api/subscription'
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
const {
  buyOffers,
  changeOffers,
  canManage,
  planLine,
  usage,
  isFull,
  waiting,
  problem,
  status,
  view,
  load,
  buy,
  change,
  manage,
} = useSubscription({
  openExternal: props.openExternal,
  client: props.client,
  now: props.now,
  settings: props.settings,
  refreshMs: props.refreshMs,
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
    <p v-if="planLine !== ''" class="subscription__plan">{{ planLine }}</p>
    <p v-if="usage !== ''" class="subscription__usage">{{ usage }}</p>
    <p v-if="isFull" class="subscription__full">{{ words.cloudFull }}</p>
    <div
      v-if="status === 'unreachable' && openSettings !== undefined"
      class="subscription__controls"
    >
      <Button type="button" variant="outline" @click="props.openSettings?.()">
        {{ words.openSettings }}
      </Button>
    </div>
    <p v-if="waiting" role="status" class="subscription__waiting">
      <strong>{{ words.waitingForPayment }}</strong>
      {{ words.waitingForPaymentText }}
    </p>
    <p v-else-if="buyOffers.length > 0" class="subscription__choose">{{ words.choosePlan }}</p>
    <div
      v-if="(!waiting && buyOffers.length > 0) || changeOffers.length > 0 || canManage"
      class="subscription__controls"
    >
      <Button
        v-for="offer in waiting ? [] : buyOffers"
        :key="`buy-${offer.id}`"
        type="button"
        variant="solid"
        @click="onBuy(offer.id)"
      >
        {{ `${words.buy} ${offer.label}` }}
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
