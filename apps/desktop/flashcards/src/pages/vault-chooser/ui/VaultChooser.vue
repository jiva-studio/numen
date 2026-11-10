<script setup lang="ts">
/**
 * The vaults of the installation to choose from, on the screen both windows open on, with what the
 * plugins say beside each one.
 */
import { VAULT_SUMMARY, type MountedView as Mounted } from '@numen/plugin-loader'
import { Spinner } from '@numen/ui'
import { WelcomePage } from '@numen/workbench'
import { MountedView } from '@/entities/tab'
import type { Vault } from '@/entities/vault'
import { VERSION } from '../lib/version'
import { WORDS } from '../words'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  vaults: readonly Vault[]
  isListing: boolean
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
    <!-- What a plugin says of each vault, such as what is due in it today. -->
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
/* Muted status text shown while the list loads. */
.vault-chooser__listing {
  display: flex;
  margin: 0;
  align-items: center;
  gap: var(--numen-inset);
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}
</style>
