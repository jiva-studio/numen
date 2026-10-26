<script setup lang="ts">
/**
 * What a vault owes today, drawn beside it on the window's list of vaults.
 *
 * Each vault's number arrives on its own. A vault whose count has not arrived
 * shows the shape that number will take; one that is being read, or could not
 * be, says so instead.
 */
import { computed } from 'vue'
import { DueCount } from '../../../ui/cards'
import type { VaultCardsDue } from '../../entities/vault'
import { getVaultSummary } from './lib/summary'
import { VAULTS_WORDS } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  /** The vault as the count has it, and nothing before the count names it. */
  vault: VaultCardsDue | null
}>()

/* --------------------------------- State ---------------------------------- */
const summary = computed(() => getVaultSummary(props.vault, VAULTS_WORDS))
</script>

<template>
  <!-- The list is where the room is shortest, so the number stands alone. -->
  <DueCount v-if="summary.kind === 'due'" :due="summary.due" is-bare />
  <span v-else class="vault-summary__detail">{{ summary.text }}</span>
</template>

<style scoped>
.vault-summary__detail {
  color: var(--numen-edge-label);
  font-size: var(--numen-text-1);
}
</style>
