<script setup lang="ts">
/** The footer of the palette: the actions bound to keys on the highlighted item, and the way to the rest. */
import { KeyCap } from '@numen/ui'
import type { PaletteShortcut } from '../lib/keys'
import type { PaletteKeys } from '@numen/ui'

defineProps<{
  /** The actions with a key binding, in display order. */
  hinted: readonly PaletteShortcut[]
  /** The keystroke that opens the action panel, as this machine reports it. */
  actionKey: PaletteKeys
  /** What the action panel is called. */
  name: string
}>()
</script>

<template>
  <footer class="palette__keys text-small text-hushed flex items-center gap-3">
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

/* The action each key triggers is shown next to its key cap. */
.palette__key,
.palette__more {
  display: inline-flex;
  align-items: center;
  gap: 0.4em;
}
</style>
