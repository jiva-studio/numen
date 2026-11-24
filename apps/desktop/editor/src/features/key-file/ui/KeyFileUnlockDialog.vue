<script setup lang="ts">
/** The question asked at the start when the keys are in a file protected by a passphrase. */
import { ref } from 'vue'
import { Button, Dialog } from '@numen/ui'
import { WORDS as words } from '../words'
import PassphraseField from './PassphraseField.vue'

/* --------------------------------- Props ---------------------------------- */
defineProps<{ problem: string; isBusy: boolean }>()

const emit = defineEmits<{
  (e: 'unlock', passphrase: string): void
  (e: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const passphrase = ref('')

/* -------------------------------- Handlers -------------------------------- */
function onSubmit(): void {
  if (passphrase.value !== '') emit('unlock', passphrase.value)
}
</script>

<template>
  <Dialog is-form :label="words.unlockTitle" :hint="words.unlockHint" @submit="onSubmit" @cancel="emit('cancel')">
    <PassphraseField
      :label="words.passphraseField"
      :value="passphrase"
      :is-disabled="isBusy"
      :is-invalid="problem !== ''"
      @change="(value) => (passphrase = value)"
    />
    <p v-if="problem !== ''" role="alert" class="key-file-unlock__problem">{{ problem }}</p>
    <template #actions>
      <Button type="button" variant="ghost" :disabled="isBusy" @click="emit('cancel')">
        {{ words.notNow }}
      </Button>
      <Button type="submit" variant="solid" :disabled="isBusy || passphrase === ''">
        {{ isBusy ? words.unlocking : words.unlockAction }}
      </Button>
    </template>
  </Dialog>
</template>

<style scoped>
.key-file-unlock__problem {
  margin: 0.5rem 0 0;
  color: var(--numen-alarm);
  font-size: var(--numen-text-2);
}
</style>
