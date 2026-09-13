<script setup lang="ts">
/**
 * The column the screen is read in: the mark and the ways in at its head, the
 * vaults under them.
 *
 * Short of the height the column needs, it stands as two, the ways in beside
 * the list.
 */
import { WelcomeLead } from '../welcome-lead'
import { WelcomeVaults } from '../welcome-vaults'
import type { Offer, VaultRow, WelcomeAction } from '../../lib/welcome'

withDefaults(
  defineProps<{
    /** What stands under the mark, telling a person which window they opened. */
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
  (event: 'takeOffer'): void
}>()

defineSlots<{
  /** What stands where the list would be while there is no list yet. */
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
      @take-offer="$emit('takeOffer')"
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
/* The column sits in the middle of whatever room there is. It is centred by its
   own margins, so a column taller than the room keeps its head inside it and
   the whole of it is scrolled to. */
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
