<script setup lang="ts">
/** The "..." button of a row and the menu it opens; choosing an item emits its id and nothing else happens here. */
import { ref, useTemplateRef } from 'vue'
import { Ellipsis, LogOut, Unplug } from '@lucide/vue'
import { Menu, measureRect, type MenuItem, type Position } from '@numen/ui'
import HeaderAction from './HeaderAction.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ label: string; items: readonly MenuItem[] }>()

const emit = defineEmits<{ (event: 'choose', id: string): void }>()

/* --------------------------------- State ---------------------------------- */
const trigger = useTemplateRef<InstanceType<typeof HeaderAction>>('trigger')
const anchor = ref<{ at: Position; wide: number } | null>(null)
const getTrigger = (): HTMLElement | null => {
  const element: unknown = trigger.value?.$el
  return element instanceof HTMLElement ? element : null
}

/* -------------------------------- Handlers -------------------------------- */
function onClose(): void {
  anchor.value = null
}

function onToggle(): void {
  if (anchor.value !== null) {
    onClose()
    return
  }
  const box = measureRect(getTrigger())
  if (box !== undefined) anchor.value = { at: { x: box.left, y: box.bottom }, wide: box.width }
}

function onChoose(id: string): void {
  onClose()
  emit('choose', id)
}
</script>

<template>
  <HeaderAction
    ref="trigger"
    :aria-label="label"
    aria-haspopup="menu"
    :aria-expanded="anchor !== null"
    @click="onToggle"
  >
    <Ellipsis :size="14" aria-hidden="true" />
  </HeaderAction>
  <Menu
    v-if="anchor"
    :items="items"
    :at="anchor.at"
    :asking="anchor.wide"
    :from="getTrigger()"
    :name="label"
    is-open
    trigger="keyboard"
    @choose="onChoose"
    @dismiss="onClose"
  >
    <template #icon="{ id }">
      <Unplug v-if="id === 'stop'" :size="14" aria-hidden="true" />
      <LogOut v-else-if="id === 'leave'" :size="14" aria-hidden="true" />
    </template>
  </Menu>
</template>
