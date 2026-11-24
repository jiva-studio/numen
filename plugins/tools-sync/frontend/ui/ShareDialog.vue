<script setup lang="ts">
/**
 * The dialog "Share <folder>": the sharing link of the person, whose name the link holds, the button that shares
 * the folder, the invitation to send once it is shared, and the people who can read this folder.
 */
import { computed } from 'vue'
import { Check, ClipboardPaste, Copy } from '@lucide/vue'
import { Button, ModalDialog, Spinner } from '@numen/ui'
import type { ShareReader } from '../api/shares'
import { getShortFingerprint } from '../lib/fingerprint'
import { isIdentityWarning, type ShareView } from '../model/shares'
import { WORDS as words } from '../words'
import SyncRow from './SyncRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: ShareView
}>()

/* --------------------------------- State ---------------------------------- */
const folderName = computed(() => props.state.target.folder.split('/').at(-1) ?? '')
const refusal = computed(() => props.state.error)
const isIdentityChange = computed(() => isIdentityWarning(props.state.error))
const hasInvitation = computed(() => props.state.invitation !== '')
const isSharing = computed(() => props.state.isBusy || props.state.share.state === 'creating')
const canShare = computed(() => !isSharing.value && props.state.code.trim() !== '')
const sharedWith = computed(() => words.sharedWith(props.state.sharedName || words.unnamed))

/* -------------------------------- Handlers -------------------------------- */
function onCodeInput(event: Event): void {
  props.state.setCode((event.target as HTMLInputElement).value).catch(props.state.fail)
}

function onShare(isConfirmed: boolean): void {
  props.state.add(isConfirmed).catch(props.state.fail)
}

function onPaste(): void {
  props.state.pasteCode().catch(props.state.fail)
}

function onRemove(person: ShareReader): void {
  props.state.removeReader(person).catch(props.state.fail)
}

function onCopyInvitation(): void {
  props.state.copyInvitation().catch(props.state.fail)
}

function onCopyIdentity(fingerprint: string): void {
  props.state.copyFingerprint(fingerprint).catch(props.state.fail)
}
</script>

<template>
  <ModalDialog
    is-form
    :label="words.shareDialogTitle(folderName)"
    :hint="words.shareDialogText"
    :step="hasInvitation ? 'shared' : 'share'"
    @submit="onShare(false)"
    @cancel="state.close"
  >
    <div v-if="hasInvitation" class="share-result">
      <p class="sync-dialog__text" role="status">
        <Check :size="14" aria-hidden="true" />
        {{ sharedWith }}
      </p>
      <textarea
        class="sync-field__input sync-field__input--code"
        readonly
        rows="3"
        :value="state.invitation"
        :aria-label="words.invitationName"
      />
      <Button variant="outline" @click="onCopyInvitation">
        <Copy :size="14" aria-hidden="true" />
        {{ words.copyInvitation }}
      </Button>
      <p class="sync-dialog__text sync-dialog__text--hushed">{{ words.acceptWhere }}</p>
    </div>
    <template v-else>
      <div class="sync-field">
        <label class="sync-field__label" for="share-code">{{ words.sharingLinkField }}</label>
        <div class="sync-field__line">
          <input
            id="share-code"
            class="sync-field__input sync-field__input--code"
            type="text"
            autocomplete="off"
            spellcheck="false"
            :value="state.code"
            :aria-invalid="state.codeError !== ''"
            @input="onCodeInput"
          />
          <Button variant="outline" @click="onPaste">
            <ClipboardPaste :size="14" aria-hidden="true" />
            {{ words.pasteCode }}
          </Button>
        </div>
        <p
          v-if="state.codeError !== ''"
          class="sync-dialog__text sync-dialog__text--alarm"
          role="alert"
        >
          {{ state.codeError }}
        </p>
        <p v-else-if="state.fingerprint !== ''" class="sync-dialog__text" role="status">
          <Check :size="14" aria-hidden="true" />
          {{ words.shareWith(state.label || state.fingerprint) }}
        </p>
        <p v-else class="sync-dialog__text sync-dialog__text--hushed">
          {{ words.sharingLinkHint }}
        </p>
      </div>
    </template>
    <p v-if="isSharing" class="sync-dialog__text" role="status">
      <Spinner class="sync-dialog__spinner" />
      {{ words.sharing }}
    </p>
    <p v-if="refusal !== ''" class="sync-dialog__text sync-dialog__text--alarm" role="alert">
      {{ refusal }}
    </p>
    <section class="share-people" :aria-label="words.peopleTitle">
      <h3 class="share-people__title">{{ words.peopleTitle }}</h3>
      <p v-if="state.people.length === 0" class="sync-dialog__text sync-dialog__text--hushed">
        {{ words.noPeople }}
      </p>
      <ul v-else class="sync-card__list">
        <SyncRow v-for="person in state.people" :key="person.fingerprint" :name="person.label">
          <span class="sync-row__detail">{{ getShortFingerprint(person.fingerprint) }}</span>
          <template #actions>
            <Button
              variant="ghost"
              size="small"
              :aria-label="words.copyIdentity"
              @click="onCopyIdentity(person.fingerprint)"
            >
              <Copy :size="14" aria-hidden="true" />
            </Button>
            <Button variant="ghost" size="small" :disabled="isSharing" @click="onRemove(person)">
              {{ words.removeAccess }}
            </Button>
          </template>
        </SyncRow>
      </ul>
    </section>
    <template #actions>
      <Button v-if="isIdentityChange" variant="outline" @click="onShare(true)">
        {{ words.shareAnyway }}
      </Button>
      <Button v-if="hasInvitation" type="button" @click="state.close">{{ words.done }}</Button>
      <template v-else>
        <Button variant="ghost" @click="state.close">{{ words.cancel }}</Button>
        <Button type="submit" :disabled="!canShare">{{ words.shareFolder }}</Button>
      </template>
    </template>
  </ModalDialog>
</template>
