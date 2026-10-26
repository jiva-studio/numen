<script setup lang="ts">
/**
 * The vaults of the installation to choose from, on the screen both windows open on, with what the
 * plugins say beside each one.
 */
import { VAULT_SUMMARY, type MountedView as Mounted } from '@numen/plugin-sdk'
import { Spinner } from '@numen/ui'
import { WelcomePage, type VaultRow } from '@numen/shell'
import { VERSION } from '@/shared/version'
import { WORDS } from '@/shared/words'
import MountedView from './MountedView.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  vaults: readonly VaultRow[]
  /** Whether the list is still being read. */
  isListing: boolean
  /** How the window draws a view a plugin declares. */
  mountView: (capability: string, element: HTMLElement, context: unknown) => Mounted
}>()

/* --------------------------------- Events --------------------------------- */
defineEmits<{ (event: 'open', vault: string): void }>()
</script>

<template>
  <WelcomePage
    name="flashcards"
    :vaults="vaults"
    :heading="WORDS.vaults"
    :version="VERSION"
    @open="$emit('open', $event)"
  >
    <!-- What a plugin says of each vault, such as what it owes today. -->
    <template #vault="{ vault }">
      <MountedView
        :capability="VAULT_SUMMARY"
        :context="{ vault: vault.id }"
        :mount-view="mountView"
      />
    </template>

    <!-- Nothing is known about the installation yet, not even which vaults it
         holds, which is the one thing the screen has to say until it is. -->
    <template v-if="isListing" #waiting>
      <p class="vault-chooser__listing" role="status">
        <Spinner />
        {{ WORDS.listing }}
      </p>
    </template>
  </WelcomePage>
</template>

<style scoped>
/* The window's own quiet voice, which is what it says while it reads the list. */
.vault-chooser__listing {
  display: flex;
  margin: 0;
  align-items: center;
  gap: var(--numen-inset);
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}
</style>
