<script setup lang="ts">
/**
 * The dialog that sets up a new identity in two steps: the twelve recovery words, shown once, and
 * the check of three of them. Nothing is kept unless the check passes.
 */
import { computed } from 'vue'
import { WORDS as words } from '../../words'
import ConfirmWords from './ConfirmWords.vue'
import IdentityDialog from './IdentityDialog.vue'
import IdentitySetupActions from './IdentitySetupActions.vue'
import type { IdentitySetupProps } from './IdentitySetup.types'
import RecoveryWords from './RecoveryWords.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<IdentitySetupProps>()

const emit = defineEmits<{
  (e: 'change', position: number, value: string): void
  (e: 'check'): void
  (e: 'back'): void
  (e: 'keep'): void
  (e: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const isWords = computed(() => props.stage === 'words')

/* -------------------------------- Handlers -------------------------------- */
function onSubmit(): void {
  if (isWords.value) emit('check')
  else if (props.isComplete) emit('keep')
}
</script>

<template>
  <IdentityDialog
    :title="isWords ? words.identitySetupTitle : words.identityCheckTitle"
    :step="stage"
    @submit="onSubmit"
    @cancel="emit('cancel')"
  >
    <template v-if="isWords">
      <p class="identity-setup__text">{{ words.identityWriteDown }}</p>
      <RecoveryWords :recovery-words="recoveryWords" />
      <p class="caution caution--card">{{ words.identityNobodyRecovers }}</p>
    </template>

    <template v-else>
      <p class="identity-setup__text">{{ words.identityConfirmHint(positions) }}</p>
      <div class="identity-setup__check">
        <ConfirmWords
          :positions="positions"
          :entered="entered"
          :wrong="wrong"
          :is-disabled="isBusy"
          @change="(position, value) => emit('change', position, value)"
          @submit="onSubmit"
        />
        <p v-if="problem !== ''" role="alert" class="identity-setup__problem">
          {{ problem }}
        </p>
      </div>
    </template>

    <IdentitySetupActions
      :stage="stage"
      :is-busy="isBusy"
      :is-complete="isComplete"
      @cancel="emit('cancel')"
      @back="emit('back')"
    />
  </IdentityDialog>
</template>

<style scoped>
.identity-setup__check {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.identity-setup__text,
.identity-setup__problem {
  margin: 0;
}

.identity-setup__problem {
  font-size: var(--numen-text-1);
  color: var(--numen-alarm);
}
</style>
