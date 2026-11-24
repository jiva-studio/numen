<script setup lang="ts">
/**
 * The dialog that restores an identity on this device from what the person enters in one field: the
 * 12 recovery words or two sheets of a recovery kit. The restore waits until the entry is whole.
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
  (e: 'restore'): void
  (e: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const isComplete = computed(() => props.reading.proof !== undefined)
</script>

<template>
  <IdentityDialog
    :title="words.identityRestoreTitle"
    :hint="words.identityRestoreHint"
    @submit="emit('restore')"
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
        {{ isBusy ? words.identityRestoring : words.identityRestoreAction }}
      </Button>
    </template>
  </IdentityDialog>
</template>
