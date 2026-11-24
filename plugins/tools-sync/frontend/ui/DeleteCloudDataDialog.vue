<script setup lang="ts">
/**
 * The question the card asks before it deletes the cloud data of the identity: what is deleted, how long the
 * server keeps it, what stays on the devices, that an active subscription is cancelled, and the account
 * fingerprint the person types to confirm. A text that is not the fingerprint shows an error under the field
 * and deletes nothing.
 */
import { ref } from 'vue'
import { ConfirmDialog } from '@numen/ui'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  account: string
  hasSubscription: boolean
}>()

const emit = defineEmits<{
  (event: 'confirm'): void
  (event: 'cancel'): void
}>()

/* --------------------------------- State ---------------------------------- */
const typed = ref('')
const wrong = ref('')

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event): void {
  typed.value = (event.target as HTMLInputElement).value
  wrong.value = ''
}

function onConfirm(): void {
  if (typed.value.trim() !== props.account) {
    wrong.value = words.deleteNameWrong
    return
  }
  emit('confirm')
}
</script>

<template>
  <ConfirmDialog
    tone="danger"
    :label="words.deleteQuestion"
    :confirm-text="words.deleteConfirm"
    :cancel-text="words.deleteCancel"
    @confirm="onConfirm"
    @cancel="emit('cancel')"
  >
    <div class="subscription__delete">
      <p>{{ words.deleteWhat }}</p>
      <p>{{ words.deleteWhen }}</p>
      <p>{{ words.deleteLocal }}</p>
      <p v-if="hasSubscription">{{ words.deleteSubscription }}</p>
      <label class="subscription__field">
        <span>{{ words.deleteTypeName(account) }}</span>
        <input
          class="subscription__input"
          type="text"
          autocomplete="off"
          spellcheck="false"
          :value="typed"
          :aria-label="words.deleteNameLabel"
          :aria-invalid="wrong !== ''"
          @input="onInput"
        />
      </label>
      <p v-if="wrong !== ''" role="alert" class="subscription__problem">{{ wrong }}</p>
    </div>
  </ConfirmDialog>
</template>
