<script setup lang="ts">
/**
 * The vaults this installation holds, under what the window calls them, and
 * the row it offers below the list.
 *
 * Each vault is opened by the letter at the end of its row, as far down the
 * list as the alphabet reaches.
 */
import { FolderRoot } from '@lucide/vue'
import { WelcomeRow } from '../welcome-row'
import { vaultLetter } from '../../lib/letters'
import type { Offer, VaultRow } from '../../lib/welcome'
import type { PaletteKeys } from '@/shared/ui/key-cap'

withDefaults(
  defineProps<{
    vaults: readonly VaultRow[]
    /** What the list is called. */
    heading: string
    /** The row below the list, where a window offers one. */
    offer?: Offer | null
  }>(),
  { offer: null },
)

defineEmits<{
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
  <section class="welcome-page__vaults">
    <h2 class="welcome-page__heading">{{ heading }}</h2>
    <!-- The room the list will fill, while the window has no rows to give
         it and something to say about that. -->
    <div v-if="!vaults.length && $slots.waiting" class="welcome-page__waiting">
      <slot name="waiting" />
    </div>
    <ul v-else class="welcome-page__list">
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
          <!-- What the window has to say about this one, drawn at the far
               end of its row. What that is belongs to the window. -->
          <slot name="vault" :vault="one" />
          <span v-if="one.detail" class="welcome-page__state">{{ one.detail }}</span>
        </WelcomeRow>
      </li>
    </ul>
    <WelcomeRow
      v-if="offer"
      :icon="offer.icon"
      :text="offer.text"
      :aside="offer.detail"
      :keys="offer.keys"
      @click="$emit('takeOffer')"
    />
  </section>
</template>

<style scoped>
.welcome-page__heading {
  margin: 0 0 0.25rem;
  padding-inline: 0.6rem;
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
  font-weight: inherit;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

.welcome-page__list {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

/* Where the rows will stand, so what is said while they are on their way is
   said in the middle of the room they will take. */
.welcome-page__waiting {
  display: flex;
  align-items: center;
  justify-content: center;
  min-block-size: 4rem;
}

/* What is true of one row of the list and not of the ones beside it, said at
   the far end of it. */
.welcome-page__state {
  flex: none;
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}

/* Where the screen stands as two columns, the heading holds its place at the
   head of this one and the offer holds its place at the foot; the rows between
   them are scrolled. */
@container (max-height: 27.55rem) and (min-width: 45.4rem) {
  .welcome-page__vaults {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-inline-size: 0;
    min-block-size: 0;
  }

  .welcome-page__list {
    min-block-size: 0;
    overflow-y: auto;
    overscroll-behavior: contain;
  }
}
</style>
