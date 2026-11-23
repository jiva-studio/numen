<script setup lang="ts">
/**
 * The dialog that restores an identity on this device from its twelve recovery words.
 */
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'
import IdentityDialog from './IdentityDialog.vue'
import RestoreWordInputs from './RestoreWordInputs.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ entered: readonly string[]; problem: string; isBusy: boolean }>()

const emit = defineEmits<{
  (e: 'change', index: number, value: string): void
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
    <p class="identity-restore__text">{{ words.identityRestoreHint }}</p>
    <div class="identity-restore__words">
      <RestoreWordInputs
        :entered="entered"
        :is-disabled="isBusy"
        @change="(index, value) => emit('change', index, value)"
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

.identity-restore__text,
.identity-restore__problem {
  margin: 0;
}

.identity-restore__problem {
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
