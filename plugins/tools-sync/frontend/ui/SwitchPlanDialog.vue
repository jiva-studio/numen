<script setup lang="ts">
/**
 * The question the card asks before it changes the plan: the plan now and the new one, what is charged at once
 * and the next payment. The amount charged now is the server's to compute, so the dialog says what it is for.
 */
import { computed } from 'vue'
import { ConfirmDialog } from '@numen/ui'
import { getOfferLine, type PlanOffer } from '../lib/subscriptionView'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  offer: PlanOffer
  currentLine: string
  renewal: string
}>()

const emit = defineEmits<{
  (event: 'confirm'): void
  (event: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const facts = computed(() => [
  { name: words.nowFact, text: props.currentLine },
  { name: words.newPlanFact, text: getOfferLine(props.offer) },
  { name: words.chargedNowFact, text: words.chargedNowText },
  ...(props.offer.price !== '' && props.renewal !== ''
    ? [{ name: words.nextPaymentFact, text: words.nextPayment(props.offer.price, props.renewal) }]
    : []),
])
</script>

<template>
  <ConfirmDialog
    :label="words.switchQuestion(offer.name)"
    :confirm-text="words.switchTo(offer.name)"
    :cancel-text="words.cancelSwitch"
    @confirm="emit('confirm')"
    @cancel="emit('cancel')"
  >
    <dl class="subscription__facts">
      <template v-for="fact in facts" :key="fact.name">
        <dt>{{ fact.name }}</dt>
        <dd>{{ fact.text }}</dd>
      </template>
    </dl>
  </ConfirmDialog>
</template>
