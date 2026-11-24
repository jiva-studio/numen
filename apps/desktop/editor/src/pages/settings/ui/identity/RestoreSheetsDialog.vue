<script setup lang="ts">
/**
 * The dialog that restores an identity from two sheets of a recovery kit by their QR codes. One camera session
 * reads the first sheet and then the second, which starts the restore. It has no field and only Cancel as a button.
 */
import { Check, Circle, LoaderCircle } from '@lucide/vue'
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
      continuous
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
        <span class="restore-slots__icon" aria-hidden="true">
          <Check v-if="sheet !== undefined" :size="14" />
          <Circle v-else :size="14" />
        </span>
        <span>{{ words.restoreSheetSlot(at + 1) }}</span>
        <span class="restore-slots__sheet">{{ sheet === undefined ? '' : words.restoreSheetNumber(sheet) }}</span>
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
  display: grid;
  grid-template-columns: 1rem 1fr 5rem;
  gap: 0.5rem;
  align-items: center;
  color: var(--numen-hushed);
}

.restore-slots__icon {
  display: flex;
  justify-content: center;
}

.restore-slots__sheet {
  text-align: end;
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
