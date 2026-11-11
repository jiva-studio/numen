<script setup lang="ts">
import type { VNode } from 'vue'
/**
 * The vaults this installation holds, under their display names, and the row
 * offered below the list.
 *
 * Each vault is opened by the letter shown at the end of its row, as many rows
 * down the list as the alphabet has letters.
 */
import { WelcomeRow } from '../welcome-row'
import { VaultList } from './vault-list'
import type { FooterEntry, VaultRow } from '../../model/welcome'

withDefaults(
  defineProps<{
    vaults: readonly VaultRow[]
    heading: string
    footer?: FooterEntry | null
  }>(),
  { footer: null },
)

defineEmits<{
  /** A vault on the list chosen. */
  (event: 'open', id: string): void
  /** The row below the list pressed. */
  (event: 'take-footer'): void
}>()

defineSlots<{
  /** Content displayed while the list is not yet loaded. */
  waiting?(): VNode[]
  /** How a vault on the list is drawn. */
  vault?(props: { vault: VaultRow }): VNode[]
}>()
</script>

<template>
  <section class="welcome-page__vaults">
    <h2 class="welcome-page__heading">{{ heading }}</h2>
    <!-- The room the list will fill, while the window has no rows to give
         it and something to say about that. -->
    <div v-if="!vaults.length && $slots.waiting" class="welcome-page__waiting">
      <slot name="waiting" />
    </div>
    <VaultList v-else :vaults="vaults" @open="$emit('open', $event)">
      <template v-if="$slots.vault" #vault="{ vault }">
        <slot name="vault" :vault="vault" />
      </template>
    </VaultList>
    <WelcomeRow
      v-if="footer"
      :icon="footer.icon"
      :text="footer.text"
      :aside="footer.detail"
      :keys="footer.keys"
      @click="$emit('take-footer')"
    />
  </section>
</template>

<style scoped>
.welcome-page__heading {
  --below: 0.25rem;
  --pad-inline: 0.6rem;

  margin: 0 0 var(--below);
  padding-inline: var(--pad-inline);
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
  font-weight: inherit;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

/* Reserves the space the rows will occupy, so the loading text is centred in
   the area they will take. */
.welcome-page__waiting {
  --least: 4rem;

  display: flex;
  align-items: center;
  justify-content: center;
  min-block-size: var(--least);
}

/* In the two-column layout, the heading stays at the top of this column and
   the footer at the bottom; the rows between them scroll. */
@container (max-height: 27.55rem) and (min-width: 45.4rem) {
  .welcome-page__vaults {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-inline-size: 0;
    min-block-size: 0;
  }
}
</style>
