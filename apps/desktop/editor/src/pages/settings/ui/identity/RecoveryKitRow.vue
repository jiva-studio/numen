<script setup lang="ts">
/**
 * The recovery kit of the identity as one row: what the kit is, and the button that saves a new kit
 * as a PDF. The row says when the kit is saved, and why it was not.
 */
import { Download } from '@lucide/vue'
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'
import SettingRow from '../setting-row/SettingRow.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ isSaved: boolean; problem: string; isBusy: boolean }>()

const emit = defineEmits<{
  (e: 'download'): void
}>()
</script>

<template>
  <SettingRow at="identity-kit" :name="words.kitTitle" :detail="words.kitDetail">
    <Button type="button" variant="outline" :disabled="isBusy" @click="emit('download')">
      <Download :size="14" aria-hidden="true" />
      {{ isBusy ? words.kitDownloading : words.kitDownload }}
    </Button>
  </SettingRow>
  <p v-if="problem !== ''" role="alert" class="recovery-kit__note recovery-kit__note--alarm">
    {{ problem }}
  </p>
  <p v-else-if="isSaved" role="status" class="recovery-kit__note">{{ words.kitSaved }}</p>
</template>

<style scoped>
.recovery-kit__note {
  margin: 0;
  padding-block: 0.5rem;
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}

.recovery-kit__note--alarm {
  color: var(--numen-alarm);
}
</style>
