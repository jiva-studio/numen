<script setup lang="ts">
/**
 * The dialog of the command "Share...": gives a folder to a reader by the code of the reader, shows the
 * invitation to send, and lists the readers of a shared folder.
 */
import { computed, onMounted } from 'vue'
import { Button } from '@numen/ui'
import { isIdentityWarning, type ShareView } from '../model/shares'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: ShareView
}>()

/* --------------------------------- State ---------------------------------- */
const refusal = computed(() => props.state.error || props.state.codeError)
const isIdentityChange = computed(() => isIdentityWarning(props.state.error))
const canShare = computed(
  () => !props.state.isBusy && props.state.code.trim() !== '' && props.state.label.trim() !== '',
)
const hasReaders = computed(() => props.state.share.readers.length > 0)
const isCreating = computed(() => props.state.share.state === 'creating')
const isActive = computed(() => props.state.share.state === 'active')

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  report(props.state.load())
})

/* -------------------------------- Handlers -------------------------------- */
function onCodeInput(event: Event): void {
  report(props.state.setCode((event.target as HTMLInputElement).value))
}

function onLabelInput(event: Event): void {
  props.state.setLabel((event.target as HTMLInputElement).value)
}

function onShare(isConfirmed: boolean): void {
  report(props.state.add(isConfirmed))
}

function onCopy(): void {
  report(navigator.clipboard.writeText(props.state.invitation))
}

/* --------------------------------- Helpers -------------------------------- */
function report(work: Promise<unknown>): void {
  work.catch((failure: unknown) => {
    props.state.fail(failure)
  })
}
</script>

<template>
  <section class="share" role="dialog" :aria-label="words.shareFolderTitle(state.target.folder)">
    <h2 class="share__title">{{ words.shareFolderTitle(state.target.folder) }}</h2>
    <p v-if="isCreating" class="share__note" role="status">{{ words.creating }}</p>
    <form class="share__form" @submit.prevent="onShare(false)">
      <label class="share__field">
        <span>{{ words.readerCode }}</span>
        <input
          class="share__input"
          type="text"
          :value="state.code"
          :aria-label="words.readerCode"
          @input="onCodeInput"
        />
      </label>
      <p v-if="state.fingerprint !== ''" class="share__fingerprint">
        <output :aria-label="words.readerFingerprint">{{ state.fingerprint }}</output>
        <span class="share__note">{{ words.compareCode }}</span>
      </p>
      <label class="share__field">
        <span>{{ words.label }}</span>
        <input
          class="share__input"
          type="text"
          :value="state.label"
          :aria-label="words.label"
          @input="onLabelInput"
        />
      </label>
      <div class="share__actions">
        <Button type="submit" variant="outline" :disabled="!canShare">{{ words.share }}</Button>
        <Button v-if="isIdentityChange" variant="outline" @click="onShare(true)">
          {{ words.shareAnyway }}
        </Button>
      </div>
    </form>
    <p v-if="refusal !== ''" class="share__note share__note--alarm" role="alert">{{ refusal }}</p>
    <div v-if="state.invitation !== ''" class="share__field">
      <textarea
        class="share__input share__input--invitation"
        readonly
        :value="state.invitation"
        :aria-label="words.invitation"
      />
      <Button variant="outline" @click="onCopy">{{ words.copy }}</Button>
      <p class="share__note">{{ words.invitationHint }}</p>
    </div>
    <ul v-if="hasReaders" class="share__readers" :aria-label="words.readers">
      <li v-for="reader in state.share.readers" :key="reader.fingerprint" class="share__reader">
        <span class="share__name">{{ reader.label }}</span>
        <span class="share__fingerprint">{{ reader.fingerprint }}</span>
        <Button variant="ghost" :disabled="state.isBusy" @click="report(state.removeReader(reader))">
          {{ words.remove }}
        </Button>
      </li>
    </ul>
    <div v-if="isActive" class="share__actions">
      <Button variant="outline" :disabled="state.isBusy" @click="report(state.stopShare())">
        {{ words.stopSharing }}
      </Button>
      <p class="share__note">{{ words.removedKeeps }}</p>
    </div>
  </section>
</template>
