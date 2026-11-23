<script setup lang="ts">
/**
 * The card of the subscription of the account on the page "Identity and licenses": the title, what the
 * server says of the subscription, and the one action, to buy it.
 */
import { onMounted } from 'vue'
import type { Failure, OpenExternalErrorCode, Result } from '@numen/plugin-sdk'
import { Button } from '@numen/ui'
import type { SubscriptionClient } from '../api/subscription'
import { useSubscription } from '../model/subscription'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  client?: SubscriptionClient | undefined
  openExternal: (url: string) => Promise<Result<null, Failure<OpenExternalErrorCode>>>
  now?: (() => Date) | undefined
}>()

/* --------------------------------- State ---------------------------------- */
const { canBuy, waiting, problem, view, load, startPurchase } = useSubscription({
  openExternal: props.openExternal,
  client: props.client,
  now: props.now,
})

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  load().catch(onFailure)
})

/* -------------------------------- Handlers -------------------------------- */
function onBuy(): void {
  startPurchase().catch(onFailure)
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
    <div v-if="canBuy" class="subscription__controls">
      <Button type="button" variant="solid" :disabled="waiting" @click="onBuy">
        {{ waiting ? words.waitingForPayment : words.buy }}
      </Button>
    </div>
    <p v-if="problem !== ''" role="alert" class="subscription__problem">{{ problem }}</p>
  </section>
</template>
