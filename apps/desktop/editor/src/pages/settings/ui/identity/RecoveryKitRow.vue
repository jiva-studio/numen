<script setup lang="ts">
/**
 * The recovery kit of the identity as one row: what the kit is, and the action that saves a new kit
 * as a PDF. The row says why a kit was not saved.
 */
import { Download } from '@lucide/vue'
import { WORDS as words } from '../../words'
import SettingRow from '../setting-row/SettingRow.vue'
import CardAction from './CardAction.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ problem: string; isBusy: boolean }>()

const emit = defineEmits<{
  (e: 'download'): void
}>()
</script>

<template>
  <SettingRow at="identity-kit" :name="words.kitTitle" :detail="words.kitDetail">
    <CardAction :is-disabled="isBusy" @click="emit('download')">
      <Download :size="14" aria-hidden="true" />
      {{ isBusy ? words.kitDownloading : words.kitDownload }}
    </CardAction>
  </SettingRow>
  <p v-if="problem !== ''" role="alert" class="recovery-kit__problem">{{ problem }}</p>
</template>

<style scoped>
.recovery-kit__problem {
  margin: 0;
  padding-block: 0.5rem;
  font-size: var(--numen-text-1);
  color: var(--numen-alarm);
}
</style>
