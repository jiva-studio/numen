<script setup lang="ts">
/**
 * The menu button of a group the person made. Its menu renames and deletes the group. It opens
 * aligned to the right edge of the button and extends to the left.
 */
import { ref } from 'vue'
import { Ellipsis } from '@lucide/vue'
import { Button, Menu, measureRect, measureWindow, type MenuItem } from '@numen/ui'
import { WORDS as words } from '../../words'

const emit = defineEmits<{
  (e: 'rename'): void
  (e: 'delete', opener: HTMLElement): void
}>()

/* --------------------------------- State ---------------------------------- */
const MARGIN = 8
const items: readonly MenuItem[] = [
  { id: 'rename', text: words.secretGroupRename },
  { id: 'delete', text: words.secretGroupDelete },
]
const anchor = ref<{ x: number; y: number; height: number } | null>(null)
const opener = ref<HTMLElement | null>(null)

/* -------------------------------- Handlers -------------------------------- */
function onOpen(event: MouseEvent): void {
  if (!(event.currentTarget instanceof HTMLElement)) return
  const box = measureRect(event.currentTarget)
  opener.value = event.currentTarget
  anchor.value = { x: box?.right ?? 0, y: box?.bottom ?? 0, height: measureWindow().height }
}

function onClose(): void {
  anchor.value = null
}

function onChoose(id: string): void {
  const from = opener.value
  onClose()
  if (id === 'rename') emit('rename')
  else if (id === 'delete' && from) emit('delete', from)
}
</script>

<template>
  <Button
    variant="outline"
    class="secret-group-action"
    size="icon"
    aria-haspopup="menu"
    :aria-expanded="anchor !== null"
    :aria-label="words.secretGroupActions"
    :title="words.secretGroupActions"
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
    :name="words.secretGroupActions"
    trigger="keyboard"
    is-open
    @choose="onChoose"
    @dismiss="onClose"
  />
</template>

<style scoped>
.secret-group-action {
  border-radius: var(--numen-radius);
}
</style>
