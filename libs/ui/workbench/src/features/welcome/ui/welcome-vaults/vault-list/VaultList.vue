<script setup lang="ts">
import type { VNode } from 'vue'
/**
 * The vault list: one vault per row, each opened by the letter at the end of its
 * row for as many rows as the alphabet covers.
 */
import { FolderRoot } from '@lucide/vue'
import { WelcomeRow } from '../../welcome-row'
import { vaultLetter } from '../../../lib/letters'
import type { VaultRow } from '../../../model/welcome'
import type { Keystroke } from '@numen/ui'

defineProps<{
  vaults: readonly VaultRow[]
}>()

defineEmits<{
  /** A vault on the list chosen. */
  (event: 'open', id: string): void
}>()

defineSlots<{
  /** How a vault on the list is drawn. */
  vault?(props: { vault: VaultRow }): VNode[]
}>()

const getVaultKeys = (pos: number, vault: VaultRow): Keystroke | undefined => {
  const letter = vaultLetter(pos)
  if (!letter || vault.isWorking) return undefined
  return { icons: [], letter }
}
</script>

<template>
  <ul class="welcome-page__list">
    <li v-for="(one, at) in vaults" :key="one.id">
      <WelcomeRow
        class="welcome-page__row--vault"
        :icon="FolderRoot"
        :text="one.name"
        :aside="one.path"
        :whole="one.path"
        :keys="getVaultKeys(at, one)"
        :is-disabled="one.isWorking"
        @click="$emit('open', one.id)"
      >
        <!-- What the window has to say about this one, drawn at the far end of
             its row. What that is belongs to the window. -->
        <slot name="vault" :vault="one" />
        <span v-if="one.detail" class="welcome-page__state">{{ one.detail }}</span>
      </WelcomeRow>
    </li>
  </ul>
</template>

<style scoped>
.welcome-page__list {
  --gap: 0.1rem;

  display: flex;
  flex-direction: column;
  gap: var(--gap);
  margin: 0;
  padding: 0;
  list-style: none;
}

/* The state that applies to one row and not to the other rows, shown at
   the far end of it. */
.welcome-page__state {
  flex: none;
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}

/* In the two-column layout, the rows between the heading and the
   footer are the one thing that scrolls. */
@container (max-height: 27.55rem) and (min-width: 45.4rem) {
  .welcome-page__list {
    min-block-size: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
  }
}
</style>
