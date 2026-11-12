<script setup lang="ts">
import type { VNode } from 'vue'
/**
 * The screen a window opens on: the mark and the name, the ways in under the
 * keystrokes that reach them, the vaults this installation holds, and what
 * build this is in the corner.
 *
 * Both windows open on it. What each of them offers is its own; the screen
 * draws the rows it is given and decides nothing.
 */
import { WelcomeColumn } from './welcome-column'
import type { FooterEntry, VaultRow, WelcomeAction } from '../model/welcome'

withDefaults(
  defineProps<{
    name?: string
    ways?: readonly WelcomeAction[]
    vaults: readonly VaultRow[]
    heading: string
    footer?: FooterEntry | null
    version?: string
  }>(),
  { name: 'numen', ways: () => [], footer: null, version: '' },
)

defineEmits<{
  /** A way chosen, by the identifier the caller gave it. */
  (event: 'run', id: string): void
  /** A vault on the list chosen. */
  (event: 'open', id: string): void
  /** The row below the list pressed. */
  (event: 'take-footer'): void
}>()

defineSlots<{
  /** Content shown in place of the list while the list is not yet available. */
  waiting?(): VNode[]
  /** How a vault on the list is drawn. */
  vault?(props: { vault: VaultRow }): VNode[]
}>()
</script>

<template>
  <div class="welcome-page">
    <WelcomeColumn
      :name="name"
      :ways="ways"
      :vaults="vaults"
      :heading="heading"
      :footer="footer"
      @run="$emit('run', $event)"
      @open="$emit('open', $event)"
      @take-footer="$emit('take-footer')"
    >
      <template v-if="$slots.waiting" #waiting>
        <slot name="waiting" />
      </template>
      <template v-if="$slots.vault" #vault="{ vault }">
        <slot name="vault" :vault="vault" />
      </template>
    </WelcomeColumn>

    <span class="welcome-page__version">{{ version }}</span>
  </div>
</template>

<style scoped>
.welcome-page {
  --glyph: 5.4rem;

  position: relative;
  display: flex;
  block-size: 100%;
  overflow: auto;
  container-type: size;
  padding: var(--numen-gutter);
  color: var(--numen-ink);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-font-size);
}

/* The build version, in the far corner outside the column. */
.welcome-page__version {
  position: absolute;
  inset-block-end: var(--numen-inset);
  inset-inline-end: var(--numen-inset);
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}
</style>
