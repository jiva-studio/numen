<script setup lang="ts">
/**
 * The dialog that restores an identity on this device from its twelve recovery words or from two
 * sheets of a recovery kit.
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
  (e: 'change', index: number, value: string): void
  (e: 'change-sheet', index: number, value: string): void
  (e: 'restore'): void
  (e: 'cancel'): void
}>()
</script>

<template>
  <IdentityDialog
    :title="words.identityRestoreTitle"
    @submit="emit('restore')"
    @cancel="emit('cancel')"
  >
    <div class="identity-restore__words">
      <RecoveryProofFields
        :kind="kind"
        :entered="entered"
        :texts="texts"
        :is-disabled="isBusy"
        @change-kind="(value) => emit('change-kind', value)"
        @change-word="(index, value) => emit('change', index, value)"
        @change-sheet="(index, value) => emit('change-sheet', index, value)"
      />
      <p v-if="problem !== ''" role="alert" class="identity-restore__problem">{{ problem }}</p>
    </div>
    <div class="identity-restore__actions">
      <Button type="button" variant="outline" :disabled="isBusy" @click="emit('cancel')">
        {{ words.identityCancel }}
      </Button>
      <Button type="submit" variant="solid" :disabled="isBusy">
        {{ isBusy ? words.identityRestoring : words.identityRestore }}
      </Button>
    </div>
  </IdentityDialog>
</template>

<style scoped>
.identity-restore__words {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.identity-restore__problem {
  margin: 0;
  font-size: var(--numen-text-1);
  color: var(--numen-alarm);
}

.identity-restore__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.375rem;
}
</style>
