<script setup lang="ts">
/**
 * The dialog "Join folder": the link pasted and what it offers, then the code to tell the owner
 * while the owner decides, then how the request ended.
 */
import { computed } from 'vue'
import { Button, Dialog, Spinner } from '@numen/ui'
import { getCodeText } from '../lib/code'
import { getShortFingerprint } from '../lib/fingerprint'
import type { AcceptanceView } from '../model/acceptance'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ view: AcceptanceView }>()

/* --------------------------------- State ---------------------------------- */
const isLink = computed(() => props.view.phase === 'link')
const isWaiting = computed(
  () => props.view.phase === 'connecting' || props.view.phase === 'waiting',
)
const canAccept = computed(() => isLink.value && props.view.preview !== undefined)
const endText = computed(() => {
  switch (props.view.phase) {
    case 'received':
      return words.receivedDone
    case 'declined':
      return words.declinedText
    case 'used':
      return words.usedText
    case 'expired':
      return words.expiredText
    case 'failed':
      return props.view.reason === '' ? words.failedText : props.view.reason
    default:
      return ''
  }
})
const problem = computed(() => props.view.error || props.view.previewError)

/* -------------------------------- Handlers -------------------------------- */
function onInput(event: Event): void {
  props.view.setLink((event.target as HTMLTextAreaElement).value).catch(props.view.fail)
}

function onAccept(): void {
  if (canAccept.value) props.view.accept().catch(props.view.fail)
}
</script>

<template>
  <Dialog
    is-form
    :label="words.acceptTitle"
    :hint="words.acceptText"
    :step="view.phase"
    @submit="onAccept"
    @cancel="view.close"
  >
    <label v-if="isLink" class="sync-field">
      <span class="sync-field__label">{{ words.linkField }}</span>
      <textarea
        class="sync-field__input sync-field__input--code"
        rows="4"
        :value="view.text"
        :aria-invalid="view.previewError !== ''"
        @input="onInput"
      />
    </label>
    <p v-if="view.preview" class="sync-dialog__text" role="status">
      {{
        words.invitedYou(getShortFingerprint(view.preview.ownerFingerprint), view.preview.folder)
      }}
    </p>
    <p v-if="view.phase === 'connecting'" class="sync-dialog__text" role="status">
      <Spinner class="sync-dialog__spinner" />
      {{ words.connecting }}
    </p>
    <div v-if="view.phase === 'waiting'" class="share-result" role="status">
      <p class="sync-dialog__text">{{ words.codeText }}</p>
      <p class="share-code">{{ getCodeText(view.code) }}</p>
      <p class="sync-dialog__text">
        <Spinner class="sync-dialog__spinner" />
        {{ words.waitingForOwner }}
      </p>
    </div>
    <p v-if="endText !== ''" class="sync-dialog__text" role="status">{{ endText }}</p>
    <p v-if="problem !== ''" class="sync-dialog__text sync-dialog__text--alarm" role="alert">
      {{ problem }}
    </p>
    <template #actions>
      <template v-if="isLink">
        <Button type="button" variant="ghost" @click="view.close">
          {{ canAccept ? words.decline : words.cancel }}
        </Button>
        <Button type="submit" :disabled="!canAccept">
          {{ canAccept ? words.accept : words.join }}
        </Button>
      </template>
      <Button v-else-if="isWaiting" type="button" variant="ghost" @click="view.close">
        {{ words.cancel }}
      </Button>
      <Button v-else type="button" @click="view.close">{{ words.done }}</Button>
    </template>
  </Dialog>
</template>
