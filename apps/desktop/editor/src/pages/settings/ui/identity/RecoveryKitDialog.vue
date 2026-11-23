<script setup lang="ts">
/**
 * The dialog that asks for the 12 recovery words or two sheets of an earlier kit before a new
 * recovery kit is saved as a PDF.
 */
import { Button } from '@numen/ui'
import type { ProofKind } from '../../model/useRecoveryProof'
import { WORDS as words } from '../../words'
import IdentityDialog from './IdentityDialog.vue'
import RecoveryProofFields from './RecoveryProofFields.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{
  kind: ProofKind
  entered: readonly string[]
  texts: readonly [string, string]
  problem: string
  isBusy: boolean
}>()

const emit = defineEmits<{
  (e: 'change-kind', kind: ProofKind): void
  (e: 'change-word', index: number, value: string): void
  (e: 'change-sheet', index: number, value: string): void
  (e: 'download'): void
  (e: 'cancel'): void
}>()
</script>

<template>
  <IdentityDialog :title="words.kitDialogTitle" @submit="emit('download')" @cancel="emit('cancel')">
    <p class="kit-dialog__text">{{ words.kitProofHint }}</p>
    <div class="kit-dialog__fields">
      <RecoveryProofFields
        :kind="kind"
        :entered="entered"
        :texts="texts"
        :is-disabled="isBusy"
        @change-kind="(value) => emit('change-kind', value)"
        @change-word="(index, value) => emit('change-word', index, value)"
        @change-sheet="(index, value) => emit('change-sheet', index, value)"
      />
      <p v-if="problem !== ''" role="alert" class="kit-dialog__problem">{{ problem }}</p>
    </div>
    <div class="kit-dialog__actions">
      <Button type="button" variant="outline" :disabled="isBusy" @click="emit('cancel')">
        {{ words.identityCancel }}
      </Button>
      <Button type="submit" variant="solid" :disabled="isBusy">
        {{ isBusy ? words.kitDownloading : words.kitDownload }}
      </Button>
    </div>
  </IdentityDialog>
</template>

<style scoped>
.kit-dialog__fields {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.kit-dialog__text,
.kit-dialog__problem {
  margin: 0;
}

.kit-dialog__problem {
  font-size: var(--numen-text-1);
  color: var(--numen-alarm);
}

.kit-dialog__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.375rem;
}
</style>
