<script setup lang="ts">
/**
 * The "..." action of the card header and the menu it opens: change the plan, manage the subscription, export
 * the identity data, and delete the cloud data, which is marked as dangerous. "Change plan" turns the menu into
 * the plans, every one with its size and price, the current one checked and not to be chosen. Choosing an item
 * emits it; nothing is done here.
 */
import { computed, defineComponent, h, markRaw, ref, useTemplateRef } from 'vue'
import { ArrowLeftRight, Check, Download, Ellipsis, Trash2, WalletCards } from '@lucide/vue'
import { CardAction, Menu, measureRect, type MenuItem, type Position } from '@numen/ui'
import type { PlanOffer } from '../lib/subscriptionView'
import { WORDS as words } from '../words'
import { getPlanItems } from './planItems'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  isExporting?: boolean
  hasAccountActions?: boolean
  canManage?: boolean
  offers?: readonly PlanOffer[]
  currentId?: string
}>()
const emit = defineEmits<{
  (event: 'choose-plan', offer: PlanOffer): void
  (event: 'manage'): void
  (event: 'export'): void
  (event: 'delete'): void
}>()

/* --------------------------------- State ---------------------------------- */
const trigger = useTemplateRef<InstanceType<typeof CardAction>>('trigger')
const getTrigger = (): HTMLElement | null => {
  const element: unknown = trigger.value?.$el
  return element instanceof HTMLElement ? element : null
}

const DANGER = markRaw(
  defineComponent({
    name: 'DeleteCloudDataItem',
    setup: () => () => h('span', { class: 'subscription__danger' }, words.deleteCloudData),
  }),
)

const isChoosingPlan = ref(false)
let isKeptOpen = false
const offers = computed(() => props.offers ?? [])

const planActions = computed<readonly MenuItem[]>(() => [
  ...(offers.value.length > 0 ? [{ id: 'change', text: words.changePlan, group: 'plan' }] : []),
  ...(props.canManage === true
    ? [{ id: 'manage', text: words.manageSubscription, group: 'plan' }]
    : []),
])
const accountActions = computed<readonly MenuItem[]>(() =>
  props.hasAccountActions === true
    ? [
        { id: 'export', text: words.exportAccountData, group: 'safe', isDisabled: props.isExporting === true },
        { id: 'delete', text: words.deleteCloudData, group: 'danger', component: DANGER },
      ]
    : [],
)
const items = computed(() =>
  isChoosingPlan.value
    ? getPlanItems(offers.value, props.currentId ?? '')
    : [...planActions.value, ...accountActions.value],
)

const anchor = ref<{ at: Position; wide: number } | null>(null)

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
  isChoosingPlan.value = false
  isKeptOpen = false
}

// The menu dismisses itself after every choice; choosing "Change plan" turns it into the plans instead.
function onDismiss(): void {
  if (isKeptOpen) {
    isKeptOpen = false
    return
  }
  onClose()
}

function onChoose(id: string): void {
  if (id === 'change') {
    isChoosingPlan.value = true
    isKeptOpen = true
    return
  }
  const offer = isChoosingPlan.value ? offers.value.find((one) => one.id === id) : undefined
  onClose()
  if (offer !== undefined) {
    if (offer.id !== props.currentId) emit('choose-plan', offer)
  } else if (id === 'manage') emit('manage')
  else if (id === 'export') emit('export')
  else if (id === 'delete') emit('delete')
}
</script>

<template>
  <CardAction
    ref="trigger"
    is-primary
    is-icon-only
    :label="words.moreActions"
    aria-haspopup="menu"
    :aria-expanded="anchor !== null"
    @click="onToggle"
  >
    <Ellipsis :size="14" aria-hidden="true" />
  </CardAction>
  <Menu
    v-if="anchor"
    :items="items"
    :at="anchor.at"
    :asking="anchor.wide"
    :from="getTrigger()"
    :current="isChoosingPlan ? currentId : undefined"
    :name="isChoosingPlan ? words.changePlanChoices : words.moreActions"
    is-open
    trigger="keyboard"
    @choose="onChoose"
    @dismiss="onDismiss"
  >
    <template #icon="{ id }">
      <Check v-if="isChoosingPlan && id === currentId" :size="14" aria-hidden="true" />
      <ArrowLeftRight v-else-if="id === 'change'" :size="14" aria-hidden="true" />
      <WalletCards v-else-if="id === 'manage'" :size="14" aria-hidden="true" />
      <Download v-else-if="id === 'export'" :size="14" aria-hidden="true" />
      <Trash2 v-else-if="id === 'delete'" :size="14" aria-hidden="true" />
    </template>
  </Menu>
</template>
