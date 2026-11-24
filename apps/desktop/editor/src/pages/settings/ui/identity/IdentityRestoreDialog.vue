<script setup lang="ts">
/**
 * The dialog that restores an identity on this device from what the person enters in one field: the
 * 12 recovery words or two sheets of a recovery kit. The QR code of each sheet can be scanned instead
 * of typed: it fills a line of the field. The restore waits until the entry is whole.
 */
import { computed, ref } from 'vue'
import { Button, QrScanner, type ScannerCamera } from '@numen/ui'
import type { ProofReading } from '../../model/readProof'
import { addScannedSheet, type ScannedSheets } from '../../model/readSheetCode'
import { WORDS as words } from '../../words'
import IdentityDialog from './IdentityDialog.vue'
import RecoveryProofField from './RecoveryProofField.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  text: string
  reading: ProofReading
  problem: string
  isBusy: boolean
  camera?: ScannerCamera
}>()

const emit = defineEmits<{
  (e: 'change', value: string): void
  (e: 'restore'): void
  (e: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const isComplete = computed(() => props.reading.proof !== undefined)
const isScanning = ref(false)
const scanned = ref<ScannedSheets | undefined>(undefined)
const scanNote = ref('')
const scanProblem = ref('')
const shownProblem = computed(() => (props.problem !== '' ? props.problem : scanProblem.value))

/* -------------------------------- Handlers -------------------------------- */
function onScanned(text: string): void {
  const step = addScannedSheet(scanned.value, text)
  if (step.next === undefined) {
    scanProblem.value = step.problem
    return
  }
  scanProblem.value = ''
  scanned.value = step.next
  scanNote.value = words.scanProgress(step.next.sheets.length)
  const lines = props.text.split('\n').filter((line) => line.trim() !== '')
  emit('change', [...lines, step.line].join('\n'))
  if (step.next.sheets.length >= 2) isScanning.value = false
}

function onUnavailable(): void {
  isScanning.value = false
  scanProblem.value = words.scanNoCamera
}

function onChange(value: string): void {
  scanProblem.value = ''
  emit('change', value)
}
</script>

<template>
  <IdentityDialog
    :title="words.identityRestoreTitle"
    :hint="words.identityRestoreHint"
    @submit="emit('restore')"
    @cancel="emit('cancel')"
  >
    <QrScanner
      v-if="isScanning"
      v-bind="camera === undefined ? {} : { camera }"
      @scanned="onScanned"
      @unavailable="onUnavailable"
    />
    <p v-if="scanNote !== '' && shownProblem === ''" role="status" class="restore-scan__note">
      {{ scanNote }}
    </p>
    <RecoveryProofField
      :text="text"
      :reading="reading"
      :problem="shownProblem"
      :is-disabled="isBusy"
      @change="onChange"
    />
    <template #actions>
      <Button type="button" variant="ghost" :disabled="isBusy" @click="emit('cancel')">
        {{ words.identityCancel }}
      </Button>
      <Button type="button" variant="outline" :disabled="isBusy" @click="isScanning = !isScanning">
        {{ isScanning ? words.scanStop : words.scanAction }}
      </Button>
      <Button type="submit" variant="solid" :disabled="isBusy || !isComplete">
        {{ isBusy ? words.identityRestoring : words.identityRestoreAction }}
      </Button>
    </template>
  </IdentityDialog>
</template>

<style scoped>
.restore-scan__note {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}
</style>
