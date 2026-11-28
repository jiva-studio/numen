<script setup lang="ts">
/**
 * The dialog that asks for the 12 recovery words or two sheets of an earlier kit before a new
 * recovery kit is saved as a PDF.
 */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import type { ProofReading } from '../../model/readProof'
import { WORDS as words } from '../../words'
import IdentityDialog from './IdentityDialog.vue'
import RecoveryProofField from './RecoveryProofField.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  text: string
  reading: ProofReading
  problem: string
  isBusy: boolean
}>()

const emit = defineEmits<{
  (e: 'change', value: string): void
  (e: 'download'): void
  (e: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const isComplete = computed(() => props.reading.proof !== undefined)
</script>

<template>
  <IdentityDialog
    :title="words.kitDialogTitle"
    :hint="words.kitProofHint"
    @submit="emit('download')"
    @cancel="emit('cancel')"
  >
    <RecoveryProofField
      :text="text"
      :reading="reading"
      :problem="problem"
      :is-disabled="isBusy"
      @change="(value) => emit('change', value)"
    />
    <template #actions>
      <Button type="button" variant="ghost" :disabled="isBusy" @click="emit('cancel')">
        {{ words.identityCancel }}
      </Button>
      <Button type="submit" variant="solid" :disabled="isBusy || !isComplete">
        {{ isBusy ? words.kitDownloading : words.kitDownload }}
      </Button>
    </template>
  </IdentityDialog>
</template>
