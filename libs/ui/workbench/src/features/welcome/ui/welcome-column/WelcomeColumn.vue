<script setup lang="ts">
import type { VNode } from 'vue'
/**
 * The column of the welcome screen: the mark and the entry points at its head, the
 * vaults below them.
 *
 * When the available height is less than the column needs, it becomes two columns:
 * the entry points beside the list.
 */
import { WelcomeLead } from '../welcome-lead'
import { WelcomeVaults } from '../welcome-vaults'
import type { FooterEntry, VaultRow, WelcomeAction } from '../../model/welcome'

withDefaults(
  defineProps<{
    name?: string
    ways?: readonly WelcomeAction[]
    vaults: readonly VaultRow[]
    heading: string
    footer?: FooterEntry | null
  }>(),
  { name: 'numen', ways: () => [], footer: null },
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
  /** Content shown in place of the list while no list is loaded. */
  waiting?(): VNode[]
  /** How a vault on the list is drawn. */
  vault?(props: { vault: VaultRow }): VNode[]
}>()
</script>

<template>
  <div class="welcome-page__column">
    <WelcomeLead :name="name" :ways="ways" @run="$emit('run', $event)" />

    <WelcomeVaults
      :vaults="vaults"
      :heading="heading"
      :footer="footer"
      @open="$emit('open', $event)"
      @take-footer="$emit('take-footer')"
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
  /* The width of a column on its own, and of the two side by side, which the
     container query below is written in. */
  --narrow: 22rem;
  --wide: 45.4rem;
  --gap: 1.4rem;

  display: flex;
  flex-direction: column;
  gap: var(--gap);
  margin: auto;
  inline-size: 100%;
  max-inline-size: var(--narrow);
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
    max-inline-size: var(--wide);
  }
}
</style>
