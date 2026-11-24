<script setup lang="ts">
/**
 * The actions of a card header as data: each shows its icon and label, the labels hide as the card
 * narrows (unless the item keeps its label), and in a card too narrow for the icons the secondary ones
 * move into a "..." menu. An item with choices is a menu button: it stays in the header and opens its choices.
 */
import { computed, ref, useTemplateRef, type Component } from 'vue'
import { Ellipsis } from '@lucide/vue'
import { measureRect } from '@/shared/api/dom'
import type { Position } from '@/shared/lib/math/geometry'
import Menu from '../menu/Menu.vue'
import type { MenuItem } from '../menu/items/item'
import CardAction from './CardAction.vue'
import type { CardActionItem } from './cardActionItem'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ items: readonly CardActionItem[]; moreLabel: string }>()

const emit = defineEmits<{ (event: 'choose', id: string): void }>()

/* --------------------------------- State ---------------------------------- */
const trigger = useTemplateRef<InstanceType<typeof CardAction>>('trigger')
const getTrigger = (): HTMLElement | null => {
  const element: unknown = trigger.value?.$el
  return element instanceof HTMLElement ? element : null
}

const anchor = ref<{ at: Position; wide: number } | null>(null)

const choice = ref<{ item: CardActionItem; at: Position; wide: number; from: HTMLElement } | null>(null)

const choices = computed<readonly MenuItem[]>(() =>
  (choice.value?.item.choices ?? []).map((one) => ({ id: one.id, text: one.label, detail: one.detail })),
)

const overflow = computed<readonly MenuItem[]>(() =>
  props.items
    .filter((item) => item.isPrimary !== true && item.choices === undefined)
    .map((item) => ({ id: item.id, text: item.label, isDisabled: item.isDisabled === true })),
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

function onPress(item: CardActionItem, event: Event): void {
  if (item.choices === undefined) {
    emit('choose', item.id)
    return
  }
  const from = event.currentTarget
  if (choice.value !== null || !(from instanceof HTMLElement)) {
    choice.value = null
    return
  }
  const box = measureRect(from)
  if (box !== undefined) choice.value = { item, at: { x: box.left, y: box.bottom }, wide: box.width, from }
}

function getChoiceIcon(id: string): Component | undefined {
  return choice.value?.item.choices?.find((one) => one.id === id)?.icon
}

function onChooseChoice(id: string): void {
  choice.value = null
  emit('choose', id)
}

function onClose(): void {
  anchor.value = null
}

function onChoose(id: string): void {
  onClose()
  emit('choose', id)
}
</script>

<template>
  <CardAction
    v-for="item in items"
    :key="item.id"
    :label="item.label"
    :is-primary="item.isPrimary === true"
    :is-secondary="item.isPrimary !== true && item.choices === undefined"
    :is-label-kept="item.isLabelKept === true"
    :has-menu="item.choices !== undefined"
    :aria-haspopup="item.choices !== undefined ? 'menu' : undefined"
    :aria-expanded="item.choices !== undefined ? choice?.item.id === item.id : undefined"
    :is-disabled="item.isDisabled === true"
    @click="onPress(item, $event)"
  >
    <component :is="item.icon" :size="14" aria-hidden="true" />
  </CardAction>
  <Menu
    v-if="choice"
    :items="choices"
    :at="choice.at"
    :asking="choice.wide"
    :from="choice.from"
    :name="choice.item.label"
    is-open
    trigger="keyboard"
    @choose="onChooseChoice"
    @dismiss="choice = null"
  >
    <template #icon="{ id }">
      <component :is="getChoiceIcon(id)" v-if="getChoiceIcon(id)" :size="14" aria-hidden="true" />
    </template>
  </Menu>
  <template v-if="overflow.length > 0">
    <CardAction
      ref="trigger"
      is-more
      :label="moreLabel"
      aria-haspopup="menu"
      :aria-expanded="anchor !== null"
      @click="onToggle"
    >
      <Ellipsis :size="14" aria-hidden="true" />
    </CardAction>
    <Menu
      v-if="anchor"
      :items="overflow"
      :at="anchor.at"
      :asking="anchor.wide"
      :from="getTrigger()"
      :name="moreLabel"
      is-open
      trigger="keyboard"
      @choose="onChoose"
      @dismiss="onClose"
    />
  </template>
</template>
