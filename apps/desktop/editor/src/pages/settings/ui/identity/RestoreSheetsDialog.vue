<script setup lang="ts">
/**
 * The dialog that restores an identity from two sheets of a recovery kit by their QR codes. One camera session
 * reads the first sheet and then the second, which starts the restore. It has no field and only Cancel as a button.
 */
import { computed } from 'vue'
import { Button, QrScanner, StageList, type ScannerCamera } from '@numen/ui'
import { getRestoreStages } from '../../model/getRestoreStages'
import { WORDS as words } from '../../words'
import IdentityDialog from './IdentityDialog.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
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

/* --------------------------------- State ---------------------------------- */
const stages = computed(() => getRestoreStages(props.slots, props.isBusy, props.problem !== ''))
</script>

<template>
  <IdentityDialog
    :title="words.identityRestoreSheets"
    :hint="words.restoreSheetsHint"
    :caution="words.restoreUnverifiedNote"
    @cancel="emit('cancel')"
  >
    <QrScanner
      is-continuous
      :rejected-message="words.identityRestoreRejectedCode"
      v-bind="camera === undefined ? {} : { camera }"
      @scanned="emit('scanned', $event)"
      @unavailable="emit('unavailable')"
    />
    <StageList :stages="stages" />
    <p v-if="problem !== ''" role="alert" class="restore-sheets__line restore-sheets__line--problem">
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
.restore-sheets__line {
  margin: 0;
  color: var(--numen-hushed);
}

.restore-sheets__line--problem {
  color: var(--numen-alarm);
}
</style>
