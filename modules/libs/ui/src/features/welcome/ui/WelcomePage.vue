<script setup lang="ts">
/**
 * The screen a window opens on: the mark and the name, the ways in under the
 * keystrokes that reach them, the vaults this installation holds, and what
 * build this is in the corner.
 *
 * Both windows open on it. What each of them offers is its own; the screen
 * draws the rows it is given and decides nothing.
 */
import { WelcomeColumn } from './welcome-column'
import type { Offer, VaultRow, WelcomeAction } from '../lib/welcome'

withDefaults(
  defineProps<{
    /**
     * What stands under the mark. Both windows wear the same glyph, and this is
     * what tells a person which of them they opened.
     */
    name?: string
    /** The ways in, above the list. A window offering none draws none. */
    ways?: readonly WelcomeAction[]
    vaults: readonly VaultRow[]
    /** What the list is called. */
    heading: string
    /** The row below the list, where a window offers one. */
    offer?: Offer | null
    version?: string
  }>(),
  { name: 'numen', ways: () => [], offer: null, version: '' },
)

defineEmits<{
  /** A way chosen, by the identifier the caller gave it. */
  (event: 'run', id: string): void
  /** A vault on the list chosen. */
  (event: 'open', id: string): void
  /** The row below the list pressed. */
  (event: 'take-offer'): void
}>()

defineSlots<{
  /** What stands where the list would be while there is no list yet. */
  waiting?(): unknown
  /** How a vault on the list is drawn. */
  vault?(props: { vault: VaultRow }): unknown
}>()
</script>

<template>
  <div class="welcome-page">
    <WelcomeColumn
      :name="name"
      :ways="ways"
      :vaults="vaults"
      :heading="heading"
      :offer="offer"
      @run="$emit('run', $event)"
      @open="$emit('open', $event)"
      @take-offer="$emit('take-offer')"
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
/* One column in the middle of the window, held to the width of a short line so
   the rows read as a list and not as a page. Short of the height that column
   needs, it stands as two. */
.welcome-page {
  /* How tall the glyph stands over the name. */
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

/* What build this is, in the far corner, where the column never reaches. */
.welcome-page__version {
  position: absolute;
  inset-block-end: var(--numen-inset);
  inset-inline-end: var(--numen-inset);
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}
</style>
