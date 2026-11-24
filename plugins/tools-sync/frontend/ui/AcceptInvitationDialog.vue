<script setup lang="ts">
/** The dialog "Accept an invitation": one field for the invitation a person sent, and the button that accepts it. */
import { computed } from 'vue'
import { Button, ModalDialog } from '@numen/ui'
import type { ReceivedView } from '../model/shares'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ view: ReceivedView }>()

const emit = defineEmits<{ (event: 'close'): void }>()

/* --------------------------------- State ---------------------------------- */
const problem = computed(() => props.view.error || props.view.invitationError)
const canAccept = computed(() => !props.view.isBusy && props.view.text.trim() !== '')

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event): void {
  props.view.setText((event.target as HTMLTextAreaElement).value).catch(props.view.fail)
}

function onAccept(): void {
  if (!canAccept.value) return
  props.view
    .receive()
    .then(() => {
      if (props.view.error === '') emit('close')
    })
    .catch(props.view.fail)
}
</script>

<template>
  <ModalDialog
    is-form
    :label="words.acceptInvitation"
    :hint="words.acceptText"
    @submit="onAccept"
    @cancel="emit('close')"
  >
    <label class="sync-field">
      <span class="sync-field__label">{{ words.invitationField }}</span>
      <textarea
        class="sync-field__input sync-field__input--code"
        rows="4"
        :value="view.text"
        :placeholder="words.invitationPlaceholder"
        :aria-invalid="problem !== ''"
        @input="onInput"
      />
    </label>
    <p v-if="view.shareName !== ''" class="sync-dialog__text" role="status">
      {{ view.shareName }}: {{ words.fromPerson(view.ownerFingerprint) }}
    </p>
    <p v-if="problem !== ''" class="sync-dialog__text sync-dialog__text--alarm" role="alert">
      {{ problem }}
    </p>
    <template #actions>
      <Button variant="ghost" @click="emit('close')">{{ words.cancel }}</Button>
      <Button type="submit" :disabled="!canAccept">{{ words.accept }}</Button>
    </template>
  </ModalDialog>
</template>
