<script setup lang="ts">
/**
 * The recovery kit of the identity as one row: what the kit is. The action that saves a new kit as a PDF
 * is in the header of the card. The row says why a kit was not saved.
 */
import { WORDS as words } from '../../words'
import SettingRow from '../setting-row/SettingRow.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ problem: string }>()
</script>

<template>
  <SettingRow at="identity-kit" :name="words.kitTitle" :detail="words.kitDetail" />
  <p v-if="problem !== ''" role="alert" class="recovery-kit__problem">{{ problem }}</p>
</template>

<style scoped>
.recovery-kit__problem {
  margin: 0;
  padding-block: 0.5rem;
  font-size: var(--numen-text-2);
  color: var(--numen-alarm);
}
</style>
