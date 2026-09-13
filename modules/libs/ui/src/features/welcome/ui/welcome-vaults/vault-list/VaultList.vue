<script setup lang="ts">
/**
 * The vaults themselves, one to a row, each opened by the letter at the end of
 * its row as far down the list as the alphabet reaches.
 */
import { FolderRoot } from '@lucide/vue'
import { WelcomeRow } from '../../welcome-row'
import { vaultLetter } from '../../../lib/letters'
import type { VaultRow } from '../../../lib/welcome'
import type { PaletteKeys } from '@/shared/ui/key-cap'

defineProps<{
  vaults: readonly VaultRow[]
}>()

defineEmits<{
  /** A vault on the list chosen. */
  (event: 'open', id: string): void
}>()

defineSlots<{
  /** How a vault on the list is drawn. */
  vault?(props: { vault: VaultRow }): unknown
}>()

/**
 * The letter a vault is opened by. Past the alphabet a vault is opened with the
 * hand and carries none, and a row still working is drawn without the letter it
 * will be opened by.
 */
const getVaultKeys = (at: number, vault: VaultRow): PaletteKeys | undefined => {
  const letter = vaultLetter(at)
  if (!letter || vault.working) return undefined
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
        :disabled="one.working"
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
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

/* What is true of one row of the list and not of the ones beside it, said at
   the far end of it. */
.welcome-page__state {
  flex: none;
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}

/* Where the screen stands as two columns, the rows between the heading and the
   offer are the one thing that scrolls. */
@container (max-height: 27.55rem) and (min-width: 45.4rem) {
  .welcome-page__list {
    min-block-size: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
  }
}
</style>
