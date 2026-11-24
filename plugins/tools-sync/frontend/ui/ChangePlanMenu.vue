<script setup lang="ts">
/**
 * The button "Change plan" of the card and the menu it opens: every plan with its size and price, the current
 * one checked, tagged "Current plan" and not to be chosen. Choosing another plan emits it; nothing changes here.
 */
import { computed, ref, useTemplateRef } from 'vue'
import { Check, ChevronDown } from '@lucide/vue'
import { Button, Menu, measureRect, type MenuItem, type Position } from '@numen/ui'
import { getOfferDetail, type PlanOffer } from '../lib/subscriptionView'
import { WORDS as words } from '../words'
import { getCurrentPlanName } from './getCurrentPlanName'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  offers: readonly PlanOffer[]
  currentId: string
  isSolid: boolean
}>()

const emit = defineEmits<{
  /** A plan other than the current one was chosen. */
  (event: 'choose', offer: PlanOffer): void
}>()

/* --------------------------------- State ---------------------------------- */
const trigger = useTemplateRef<InstanceType<typeof Button>>('trigger')
const getTrigger = (): HTMLElement | null => {
  const element: unknown = trigger.value?.$el
  return element instanceof HTMLElement ? element : null
}

/** Where the menu is drawn and how wide its button is, and nothing while it is closed. */
const anchor = ref<{ at: Position; wide: number } | null>(null)

const items = computed<readonly MenuItem[]>(() =>
  props.offers.map((offer) => ({
    id: offer.id,
    text: offer.name,
    detail: getOfferDetail(offer),
    ...(offer.id === props.currentId
      ? { isDisabled: true, component: getCurrentPlanName(offer.name) }
      : {}),
  })),
)

/* -------------------------------- Handlers -------------------------------- */
function onToggle(): void {
  if (anchor.value !== null) {
    onClose()
    return
  }
  const box = measureRect(getTrigger())
  if (box !== undefined) anchor.value = { at: { x: box.left, y: box.bottom }, wide: box.width }
}

function onClose(): void {
  anchor.value = null
}

function onChoose(id: string): void {
  onClose()
  const offer = props.offers.find((one) => one.id === id)
  if (offer !== undefined && offer.id !== props.currentId) emit('choose', offer)
}
</script>

<template>
  <Button
    ref="trigger"
    type="button"
    class="subscription__change"
    :variant="isSolid ? 'solid' : 'outline'"
    aria-haspopup="menu"
    :aria-expanded="anchor !== null"
    @click="onToggle"
  >
    {{ words.changePlan }}
    <ChevronDown :size="14" aria-hidden="true" />
  </Button>
  <Menu
    v-if="anchor"
    :items="items"
    :at="anchor.at"
    :asking="anchor.wide"
    :from="getTrigger()"
    :current="currentId"
    :name="words.changePlanChoices"
    is-open
    trigger="keyboard"
    @choose="onChoose"
    @dismiss="onClose"
  >
    <template #icon="{ id }">
      <Check v-if="id === currentId" :size="14" aria-hidden="true" />
    </template>
  </Menu>
</template>
