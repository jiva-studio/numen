<script setup lang="ts">
/** The foot of the palette: what a key reaches on the lit item, and the way to the rest. */
import { KeyCap } from '@/shared/ui/key-cap'
import type { PaletteShortcut } from '../lib/keys'
import type { PaletteKeys } from '@/shared/ui/key-cap'

defineProps<{
  /** The actions a key reaches, in the order they are offered. */
  hinted: readonly PaletteShortcut[]
  /** The keystroke that opens the action panel, as this machine reports it. */
  actionKey: PaletteKeys
  /** What the action panel is called. */
  name: string
}>()
</script>

<template>
  <footer class="palette__keys flex items-center gap-3 text-small text-hushed">
    <span v-for="one in hinted" :key="one.action.id" class="palette__key" data-palette="key">
      <KeyCap v-if="one.key" :keys="one.key" />
      {{ one.action.text }}
    </span>
    <span class="palette__more ml-auto" data-palette="more">
      <KeyCap :keys="actionKey" />
      {{ name }}
    </span>
  </footer>
</template>

<style scoped>
.palette__keys {
  padding: var(--numen-inset) var(--numen-inset-wide);
  border-block-start: var(--numen-stroke) solid var(--numen-panel-border);
}

/* What a key reaches stands beside the cap that reaches it. */
.palette__key,
.palette__more {
  display: inline-flex;
  align-items: center;
  gap: 0.4em;
}
</style>
