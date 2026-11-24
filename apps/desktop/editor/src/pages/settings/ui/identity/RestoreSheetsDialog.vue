<script setup lang="ts">
/**
 * The dialog that restores an identity from two sheets of a recovery kit by their QR codes. The camera
 * stays on: each sheet fills a slot, and the second one starts the restore. It has no field.
 */
import { Check, LoaderCircle } from '@lucide/vue'
import { Button, QrScanner, type ScannerCamera } from '@numen/ui'
import { WORDS as words } from '../../words'
import IdentityDialog from './IdentityDialog.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  slots: readonly (number | undefined)[]
  note: string
  problem: string
  isBusy: boolean
  camera?: ScannerCamera
}>()

const emit = defineEmits<{
  (e: 'scanned', text: string): void
  (e: 'unavailable'): void
  (e: 'retry'): void
  (e: 'cancel'): void
}>()
</script>

<template>
  <IdentityDialog
    :title="words.identityRestoreSheets"
    :hint="words.restoreSheetsHint"
    @cancel="emit('cancel')"
  >
    <QrScanner
      v-bind="camera === undefined ? {} : { camera }"
      @scanned="emit('scanned', $event)"
      @unavailable="emit('unavailable')"
    />
    <ul class="restore-slots">
      <li
        v-for="(sheet, at) in slots"
        :key="at"
        class="restore-slots__slot"
        :class="{ 'restore-slots__slot--done': sheet !== undefined }"
      >
        <Check v-if="sheet !== undefined" :size="14" aria-hidden="true" />
        {{ sheet === undefined ? words.restoreSheetWaiting(at + 1) : words.restoreSheetScanned(sheet) }}
      </li>
    </ul>
    <p v-if="isBusy" role="status" class="restore-sheets__line">
      <LoaderCircle :size="14" aria-hidden="true" />
      {{ words.restoringNow }}
    </p>
    <p v-else-if="problem !== ''" role="alert" class="restore-sheets__line restore-sheets__line--problem">
      {{ problem }}
    </p>
    <p v-else-if="note !== ''" role="alert" class="restore-sheets__line restore-sheets__line--problem">
      {{ note }}
    </p>
    <template #actions>
      <Button type="button" variant="ghost" :disabled="isBusy" @click="emit('cancel')">
        {{ words.identityCancel }}
      </Button>
      <Button v-if="problem !== ''" type="button" variant="solid" @click="emit('retry')">
        {{ words.restoreTryAgain }}
      </Button>
    </template>
  </IdentityDialog>
</template>

<style scoped>
.restore-slots {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.restore-slots__slot {
  display: flex;
  gap: 0.375rem;
  align-items: center;
  color: var(--numen-hushed);
}

.restore-slots__slot--done {
  color: var(--numen-ink);
}

.restore-sheets__line {
  display: flex;
  gap: 0.375rem;
  align-items: center;
  margin: 0;
  color: var(--numen-hushed);
}

.restore-sheets__line--problem {
  color: var(--numen-alarm);
}
</style>
