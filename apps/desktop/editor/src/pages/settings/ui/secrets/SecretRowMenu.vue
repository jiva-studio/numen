<script setup lang="ts">
/**
 * The menu button at the end of a saved secret row. Its menu moves the secret up or down, when the
 * group holds more than one, and deletes it. It opens aligned to the right edge of the button and
 * extends to the left.
 */
import { computed, ref } from 'vue'
import { Ellipsis } from '@lucide/vue'
import { Button, Menu, measureRect, measureWindow, type MenuItem } from '@numen/ui'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  isFirst: boolean
  isLast: boolean
  canMove: boolean
}>()

const emit = defineEmits<{
  (e: 'move-up'): void
  (e: 'move-down'): void
  (e: 'delete', opener: HTMLElement): void
}>()

/* --------------------------------- State ---------------------------------- */
const MARGIN = 8
const anchor = ref<{ x: number; y: number; width: number; height: number } | null>(null)
const opener = ref<HTMLElement | null>(null)
const items = computed<readonly MenuItem[]>(() => [
  ...(props.canMove
    ? [
        { id: 'up', text: words.secretMoveUp, isDisabled: props.isFirst },
        { id: 'down', text: words.secretMoveDown, isDisabled: props.isLast },
      ]
    : []),
  { id: 'delete', text: words.secretDeleteItem },
])

/* -------------------------------- Handlers -------------------------------- */
function onOpen(event: MouseEvent): void {
  if (!(event.currentTarget instanceof HTMLElement)) return
  const box = measureRect(event.currentTarget)
  opener.value = event.currentTarget
  anchor.value = {
    x: box?.right ?? 0,
    y: box?.bottom ?? 0,
    width: 0,
    height: measureWindow().height,
  }
}

function onClose(): void {
  anchor.value = null
}

function onChoose(id: string): void {
  const from = opener.value
  onClose()
  if (id === 'up') emit('move-up')
  else if (id === 'down') emit('move-down')
  else if (id === 'delete' && from) emit('delete', from)
}
</script>

<template>
  <Button
    variant="outline"
    class="secret-action"
    size="icon"
    aria-haspopup="menu"
    :aria-expanded="anchor !== null"
    :aria-label="words.secretActions"
    :title="words.secretActions"
    @click="onOpen"
  >
    <Ellipsis :size="14" aria-hidden="true" />
  </Button>
  <Menu
    v-if="anchor"
    :items="items"
    :at="{ x: anchor.x, y: anchor.y }"
    :viewport="{ width: anchor.x + MARGIN, height: anchor.height }"
    :margin="MARGIN"
    :from="opener"
    :name="words.secretActions"
    trigger="keyboard"
    is-open
    @choose="onChoose"
    @dismiss="onClose"
  />
</template>

<style scoped>
.secret-action {
  border-radius: var(--numen-radius);
}
</style>
