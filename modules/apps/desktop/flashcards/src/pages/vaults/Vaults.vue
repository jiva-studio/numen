<script setup lang="ts">
/**
 * The front door: every vault the installation holds, and what each owes today.
 *
 * It is the screen the editor opens on, drawn from the same component. The
 * screen knows nothing about cards; what stands at the end of a row is put
 * there from here.
 *
 * The list is on screen before any of it is counted, and each vault's number
 * arrives on its own. A vault whose count has not arrived shows the shape that
 * number will take and is not opened until it has one.
 */
import { computed } from 'vue'
import { DueCount, Spinner, WelcomePage } from '@numen/ui'
import type { VaultCardsDue } from '@/entities/vault'
import { getDueByVault, getVaultRows } from './lib/rows'
import { VAULTS_WORDS } from './words'

const props = defineProps<{
  vaults: readonly VaultCardsDue[]
  counting: boolean
  version: string
}>()

defineEmits<{ (event: 'choose', vault: string): void }>()

/** Each vault as a row of the list, and how many cards it has waiting. */
const listed = computed(() => getVaultRows(props.vaults, VAULTS_WORDS))

const waiting = computed(() => getDueByVault(props.vaults))
</script>

<template>
  <WelcomePage
    name="flashcards"
    :vaults="listed"
    :heading="VAULTS_WORDS.heading"
    :version="version"
    @open="$emit('choose', $event)"
  >
    <!-- The list is where the room is shortest, so the number stands alone. -->
    <template #vault="{ vault }">
      <DueCount v-if="waiting.has(vault.id)" :due="waiting.get(vault.id) ?? null" bare />
    </template>

    <!-- Nothing is known about the installation yet, not even which vaults it
         holds, which is the one thing the screen has to say until it is. -->
    <template v-if="counting" #waiting>
      <p class="vaults__counting" role="status">
        <Spinner />
        {{ VAULTS_WORDS.counting }}
      </p>
    </template>
  </WelcomePage>
</template>

<style scoped>
/* The screen's own quiet voice, which is what everything it says beside a row
   is set in. */
.vaults__counting {
  display: flex;
  margin: 0;
  align-items: center;
  gap: var(--numen-inset);
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}
</style>
