<script setup lang="ts">
/**
 * The question asked on a computer without a system keychain: keep the keys in a file protected by a
 * passphrase, or in a plain file. The passphrase asks to be repeated; nothing is sent before it matches.
 */
import { Button, Dialog } from '@numen/ui'
import { useKeyFileChoice } from '../model/useKeyFileChoice'
import { WORDS as words } from '../words'
import PassphraseField from './PassphraseField.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ problem: string; isBusy: boolean }>()

const emit = defineEmits<{
  (e: 'choose', mode: 'passphrase' | 'plaintext', passphrase: string): void
  (e: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const { mode, passphrase, confirmation, isComplete, isMismatch } = useKeyFileChoice()

/* -------------------------------- Handlers -------------------------------- */
function onSubmit(): void {
  if (!isComplete.value) return
  emit('choose', mode.value, mode.value === 'passphrase' ? passphrase.value : '')
}
</script>

<template>
  <Dialog
    is-form
    :label="words.chooseTitle"
    :hint="words.chooseHint"
    :step="mode"
    @submit="onSubmit"
    @cancel="emit('cancel')"
  >
    <fieldset class="key-file-choice" :disabled="isBusy">
      <legend class="key-file-choice__legend">{{ words.choiceLabel }}</legend>
      <label class="key-file-choice__card" :class="{ 'key-file-choice__card--on': mode === 'passphrase' }">
        <input v-model="mode" type="radio" name="key-file-mode" value="passphrase" />
        <span class="key-file-choice__text">
          <span class="key-file-choice__title">{{ words.passphraseTitle }}</span>
          <span class="key-file-choice__detail">{{ words.passphraseRule }}</span>
        </span>
      </label>
      <div v-if="mode === 'passphrase'" class="key-file-choice__fields">
        <PassphraseField
          :label="words.passphraseField"
          :value="passphrase"
          :is-disabled="isBusy"
          @change="(value) => (passphrase = value)"
        />
        <PassphraseField
          :label="words.confirmField"
          :value="confirmation"
          :is-disabled="isBusy"
          :is-invalid="isMismatch"
          @change="(value) => (confirmation = value)"
        />
        <p v-if="isMismatch" role="alert" class="key-file-choice__problem">{{ words.mismatch }}</p>
      </div>
      <label class="key-file-choice__card" :class="{ 'key-file-choice__card--on': mode === 'plaintext' }">
        <input v-model="mode" type="radio" name="key-file-mode" value="plaintext" />
        <span class="key-file-choice__text">
          <span class="key-file-choice__title">{{ words.plaintextTitle }}</span>
          <span class="key-file-choice__detail">{{ words.plaintextWarning }}</span>
        </span>
      </label>
    </fieldset>
    <p v-if="problem !== ''" role="alert" class="key-file-choice__problem">{{ problem }}</p>
    <template #actions>
      <Button type="button" variant="ghost" :disabled="isBusy" @click="emit('cancel')">
        {{ words.cancel }}
      </Button>
      <Button type="submit" variant="solid" :disabled="isBusy || !isComplete">
        {{ isBusy ? words.saving : words.continue }}
      </Button>
    </template>
  </Dialog>
</template>

<style scoped>
.key-file-choice {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin: 0;
  padding: 0;
  border: none;
}

.key-file-choice__legend {
  position: absolute;
  inline-size: 1px;
  block-size: 1px;
  overflow: hidden;
  clip-path: inset(50%);
}

.key-file-choice__card {
  display: flex;
  gap: 0.625rem;
  align-items: flex-start;
  padding: 0.625rem 0.75rem;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  cursor: pointer;
}

.key-file-choice__card--on {
  background: var(--numen-selection);
  border-color: var(--numen-accent);
}

.key-file-choice__text {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.key-file-choice__title {
  font-weight: 500;
}

.key-file-choice__detail {
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}

.key-file-choice__fields {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding-inline-start: 1.5rem;
}

.key-file-choice__problem {
  margin: 0;
  color: var(--numen-alarm);
  font-size: var(--numen-text-2);
}
</style>
