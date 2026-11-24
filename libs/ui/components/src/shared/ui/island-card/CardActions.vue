<script setup lang="ts">
/**
 * The actions of a card header as data: each shows its icon and label, the labels hide as the card
 * narrows, and in a card too narrow for the icons the secondary ones move into a "..." menu.
 */
import { computed, ref, useTemplateRef } from 'vue'
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

const overflow = computed<readonly MenuItem[]>(() =>
  props.items
    .filter((item) => item.isPrimary !== true)
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
    :is-secondary="item.isPrimary !== true"
    :is-disabled="item.isDisabled === true"
    @click="emit('choose', item.id)"
  >
    <component :is="item.icon" :size="14" aria-hidden="true" />
  </CardAction>
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
