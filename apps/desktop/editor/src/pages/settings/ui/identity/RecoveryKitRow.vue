<script setup lang="ts">
/**
 * The recovery kit of the identity as one row: what the kit is, and the action that saves a new kit
 * as a PDF at the right of the row. The action is in the row, not in the header of the card, because
 * the row is about one object. The row says why a kit was not saved.
 */
import { Download } from '@lucide/vue'
import { Button, Spinner } from '@numen/ui'
import { WORDS as words } from '../../words'
import SettingRow from '../setting-row/SettingRow.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ problem: string; isBusy: boolean }>()

const emit = defineEmits<{
  (e: 'download'): void
}>()
</script>

<template>
  <SettingRow at="identity-kit" :name="words.kitTitle" :detail="words.kitDetail">
    <Button
      type="button"
      variant="ghost"
      size="icon"
      class="recovery-kit__download"
      :aria-label="isBusy ? words.kitDownloading : words.kitDialogTitle"
      :title="isBusy ? words.kitDownloading : words.kitDialogTitle"
      :disabled="isBusy"
      @click="emit('download')"
    >
      <Spinner v-if="isBusy" />
      <Download v-else :size="16" aria-hidden="true" />
    </Button>
  </SettingRow>
  <p v-if="problem !== ''" role="alert" class="recovery-kit__problem">{{ problem }}</p>
</template>

<style scoped>
.recovery-kit__download {
  justify-self: end;
}

.recovery-kit__problem {
  margin: 0;
  padding-block: 0.5rem;
  font-size: var(--numen-text-2);
  color: var(--numen-alarm);
}
</style>
