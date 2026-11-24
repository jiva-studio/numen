<script setup lang="ts">
/**
 * The dialog "Share <folder>" of the invitation: the link to send with its validity and the wait for the
 * person, the request to read with its code and the buttons that allow or decline it, and the people who
 * can read the folder once one is allowed.
 */
import { computed } from 'vue'
import { Check, Copy } from '@lucide/vue'
import { Button, ModalDialog, Spinner } from '@numen/ui'
import { getShortFingerprint } from '../lib/fingerprint'
import type { InvitationView } from '../model/invitation'
import { WORDS as words } from '../words'
import SyncRow from './SyncRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: InvitationView }>()

/* --------------------------------- State ---------------------------------- */
const folderName = computed(() => props.state.target.folder.split('/').at(-1) ?? '')
const hasRequest = computed(() => props.state.request !== undefined)
const isAsking = computed(() => props.state.phase === 'waiting' && hasRequest.value)
const isWaiting = computed(() => props.state.phase === 'waiting' && !hasRequest.value)
const isEnded = computed(() => props.state.phase === 'approved')

/* -------------------------------- Handlers -------------------------------- */
function onCopy(): void {
  props.state.copyLink().catch(props.state.fail)
}

function onRevoke(): void {
  props.state.revoke().catch(props.state.fail)
}

function onAllow(): void {
  props.state.approve().catch(props.state.fail)
}

function onDecline(): void {
  props.state.decline().catch(props.state.fail)
}
</script>

<template>
  <ModalDialog
    :label="words.shareDialogTitle(folderName)"
    :hint="words.shareDialogText"
    :step="state.phase"
    @cancel="state.close"
  >
    <p v-if="state.phase === 'starting'" class="sync-dialog__text" role="status">
      <Spinner class="sync-dialog__spinner" />
      {{ words.preparing }}
    </p>
    <div v-if="isWaiting || isAsking" class="share-result">
      <textarea
        class="sync-field__input sync-field__input--code"
        readonly
        rows="3"
        :value="state.link"
        :aria-label="words.shareLinkField"
      />
      <Button variant="outline" @click="onCopy">
        <Copy :size="14" aria-hidden="true" />
        {{ words.copyLinkButton }}
      </Button>
      <p class="sync-dialog__text sync-dialog__text--hushed">{{ words.invitationValidity }}</p>
    </div>
    <p v-if="isWaiting" class="sync-dialog__text" role="status">
      <Spinner class="sync-dialog__spinner" />
      {{ words.waitingForPerson }}
    </p>
    <div v-if="isAsking && state.request" class="share-result" role="alert">
      <p class="sync-dialog__text">
        {{ words.wantsToRead(state.request.name, state.request.folderName) }}
      </p>
      <p class="sync-dialog__text">{{ words.confirmationCode(state.request.code) }}</p>
    </div>
    <p v-if="isEnded" class="sync-dialog__text" role="status">
      <Check :size="14" aria-hidden="true" />
      {{ words.sharedWithName(state.people.at(-1)?.name ?? '') }}
    </p>
    <p v-if="state.phase === 'declined'" class="sync-dialog__text" role="status">
      {{ words.declinedText }}
    </p>
    <p v-if="state.phase === 'expired'" class="sync-dialog__text" role="status">
      {{ words.expiredText }}
    </p>
    <p v-if="state.error !== ''" class="sync-dialog__text sync-dialog__text--alarm" role="alert">
      {{ state.error }}
    </p>
    <section v-if="state.people.length > 0" class="share-people" :aria-label="words.peopleTitle">
      <h3 class="share-people__title">{{ words.peopleTitle }}</h3>
      <ul class="sync-card__list">
        <SyncRow v-for="person in state.people" :key="person.fingerprint" :name="person.name">
          <span class="sync-row__detail">{{ getShortFingerprint(person.fingerprint) }}</span>
        </SyncRow>
      </ul>
    </section>
    <template #actions>
      <template v-if="isAsking">
        <Button variant="ghost" :disabled="state.isBusy" @click="onDecline">
          {{ words.decline }}
        </Button>
        <Button :disabled="state.isBusy" @click="onAllow">{{ words.allow }}</Button>
      </template>
      <template v-else-if="isWaiting">
        <Button variant="ghost" :disabled="state.isBusy" @click="onRevoke">
          {{ words.revoke }}
        </Button>
        <Button @click="state.close">{{ words.done }}</Button>
      </template>
      <Button v-else @click="state.close">{{ words.done }}</Button>
    </template>
  </ModalDialog>
</template>
