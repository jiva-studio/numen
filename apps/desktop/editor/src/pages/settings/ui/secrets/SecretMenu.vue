<script setup lang="ts">
/**
 * An ellipsis button that opens a menu of items. The menu opens aligned to the right edge of the
 * button and extends to the left. A chosen item is emitted with its id and the button, so the
 * caller can return the focus to it.
 */
import { ref } from 'vue'
import { Ellipsis } from '@lucide/vue'
import { Menu, measureRect, measureWindow, type MenuItem } from '@numen/ui'
import SecretActionButton from './SecretActionButton.vue'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{
  label: string
  items: readonly MenuItem[]
}>()

const emit = defineEmits<{
  (e: 'choose', id: string, opener: HTMLElement): void
}>()

/* --------------------------------- State ---------------------------------- */
const MARGIN = 8
const anchor = ref<{ x: number; y: number; height: number } | null>(null)
const opener = ref<HTMLElement | null>(null)

/* -------------------------------- Handlers -------------------------------- */
function onOpen(event: MouseEvent): void {
  if (!(event.currentTarget instanceof HTMLElement)) return
  const box = measureRect(event.currentTarget)
  opener.value = event.currentTarget
  anchor.value = { x: box.right, y: box.bottom, height: measureWindow().height }
}

function onClose(): void {
  anchor.value = null
}

function onChoose(id: string): void {
  const from = opener.value
  onClose()
  if (from) emit('choose', id, from)
}
</script>

<template>
  <SecretActionButton
    aria-haspopup="menu"
    :aria-expanded="anchor !== null"
    :aria-label="label"
    :title="label"
    @click="onOpen"
  >
    <Ellipsis :size="14" aria-hidden="true" />
  </SecretActionButton>
  <Menu
    v-if="anchor"
    :items="items"
    :at="{ x: anchor.x, y: anchor.y }"
    :viewport="{ width: anchor.x + MARGIN, height: anchor.height }"
    :margin="MARGIN"
    :from="opener"
    :name="label"
    trigger="keyboard"
    is-open
    @choose="onChoose"
    @dismiss="onClose"
  />
</template>
