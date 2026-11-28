<script setup lang="ts">
/**
 * The question asked on a computer without a system keychain: keep the keys in a file protected by a
 * passphrase, or in a plain file. The passphrase asks to be repeated; nothing is sent before it matches.
 */
import { Button, Dialog } from '@numen/ui'
import { useKeyFileChoice } from '../model/useKeyFileChoice'
import { WORDS as words } from '../words'
import KeyFileOption from './KeyFileOption.vue'
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
      <KeyFileOption
        value="passphrase"
        :is-on="mode === 'passphrase'"
        :title="words.passphraseTitle"
        @pick="mode = $event"
      />
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
      <KeyFileOption
        value="plaintext"
        :is-on="mode === 'plaintext'"
        :title="words.plaintextTitle"
        :detail="words.plaintextWarning"
        @pick="mode = $event"
      />
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
  inline-size: 0.0625rem;
  block-size: 0.0625rem;
  overflow: hidden;
  clip-path: inset(50%);
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
