<script setup lang="ts">
/**
 * The column of the welcome screen: the mark and the entry points at its head, the
 * vaults below them.
 *
 * When the available height is less than the column needs, it becomes two columns:
 * the entry points beside the list.
 */
import { WelcomeLead } from '../welcome-lead'
import { WelcomeVaults } from '../welcome-vaults'
import type { Offer, VaultRow, WelcomeAction } from '../../lib/welcome'

withDefaults(
  defineProps<{
    /** Text placed under the mark, identifying which window the user opened. */
    name?: string
    /** The ways in, above the list. */
    ways?: readonly WelcomeAction[]
    vaults: readonly VaultRow[]
    /** What the list is called. */
    heading: string
    /** The row below the list, where a window offers one. */
    offer?: Offer | null
  }>(),
  { name: 'numen', ways: () => [], offer: null },
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
  /** Content shown in place of the list while no list is loaded. */
  waiting?(): unknown
  /** How a vault on the list is drawn. */
  vault?(props: { vault: VaultRow }): unknown
}>()
</script>

<template>
  <div class="welcome-page__column">
    <WelcomeLead :name="name" :ways="ways" @run="$emit('run', $event)" />

    <WelcomeVaults
      :vaults="vaults"
      :heading="heading"
      :offer="offer"
      @open="$emit('open', $event)"
      @take-offer="$emit('take-offer')"
    >
      <template v-if="$slots.waiting" #waiting>
        <slot name="waiting" />
      </template>
      <template v-if="$slots.vault" #vault="{ vault }">
        <slot name="vault" :vault="vault" />
      </template>
    </WelcomeVaults>
  </div>
</template>

<style scoped>
.welcome-page__column {
  display: flex;
  flex-direction: column;
  gap: 1.4rem;
  margin: auto;
  inline-size: 100%;
  max-inline-size: 22rem;
}

/* Short of the height the one column takes — the mark over the name, six ways
   in, the heading and two vaults — the ways in and the vaults stand side by
   side, each column the width of a short line. The list is the one thing here
   with no end to it, and the one thing that scrolls. */
@container (max-height: 27.55rem) and (min-width: 45.4rem) {
  .welcome-page__column {
    flex-direction: row;
    margin-block: 0;
    block-size: 100%;
    max-inline-size: 45.4rem;
  }
}
</style>
