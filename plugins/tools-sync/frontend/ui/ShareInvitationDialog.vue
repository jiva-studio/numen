<script setup lang="ts">
/**
 * The dialog "Share folder": the link to send with its validity, the wait for a device, the request to read
 * with its code and the buttons that allow or decline it, and the devices that read the folder.
 */
import { computed, ref, watch } from 'vue'
import { Copy } from '@lucide/vue'
import { Button, Dialog, Spinner } from '@numen/ui'
import { getCodeText } from '../lib/code'
import { getShortFingerprint } from '../lib/fingerprint'
import type { InvitationView } from '../model/invitation'
import { WORDS as words } from '../words'
import SyncRow from './SyncRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: InvitationView }>()

/* --------------------------------- State ---------------------------------- */
const folderName = computed(() => props.state.name)
const isAsking = computed(() => props.state.request !== undefined)
const isWaiting = computed(() => props.state.isLinkAlive && !isAsking.value)
const step = computed(() => (isAsking.value ? 'asking' : props.state.isLinkAlive ? 'link' : 'done'))
const isConfirmed = ref(false)
watch(() => props.state.request?.requestId, () => {
  isConfirmed.value = false
})
const getDevice = (name: string, fingerprint: string): string =>
  words.deviceLabel(name, getShortFingerprint(fingerprint))

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
  <Dialog :label="words.shareDialogTitle" :step="step" @cancel="state.close">
    <p v-if="state.isStarting" class="sync-dialog__text" role="status">
      <Spinner class="sync-dialog__spinner" />
      {{ words.preparing }}
    </p>
    <div v-if="state.isLinkAlive" class="share-result">
      <textarea
        class="sync-field__input sync-field__input--code"
        readonly
        rows="3"
        :value="state.link"
        :aria-label="words.shareLinkField"
      />
      <p class="sync-dialog__text sync-dialog__text--hushed">
        {{ words.validUntil(state.getTimeLeft()) }}
      </p>
    </div>
    <p v-if="isWaiting" class="sync-dialog__text" role="status">
      <Spinner class="sync-dialog__spinner" />
      {{ words.waitingForPerson }}
    </p>
    <div v-if="state.request" class="share-result" role="alert">
      <h3 class="share-people__title">{{ words.joinRequest }}</h3>
      <p class="sync-dialog__text">
        {{
          words.wantsToJoin(
            getDevice(state.request.deviceName, state.request.fingerprint),
            folderName,
          )
        }}
        {{ words.confirmationCode(getCodeText(state.request.code)) }}
      </p>
      <p class="sync-dialog__text">{{ words.requestFingerprint(state.request.fingerprint) }}</p>
      <p class="sync-dialog__text sync-dialog__text--hushed">{{ words.fingerprintNote }}</p>
      <label class="sync-dialog__text">
        <input v-model="isConfirmed" type="checkbox" />
        {{ words.fingerprintConfirm }}
      </label>
      <p class="sync-dialog__text sync-dialog__text--hushed">{{ words.readOnlyHint }}</p>
    </div>
    <p v-if="state.error !== ''" class="sync-dialog__text sync-dialog__text--alarm" role="alert">
      {{ state.error }}
    </p>
    <section v-if="state.people.length > 0" class="share-people" :aria-label="words.peopleTitle">
      <h3 class="share-people__title">{{ words.peopleTitle }}</h3>
      <ul class="sync-card__list">
        <SyncRow
          v-for="person in state.people"
          :key="person.fingerprint"
          :name="getDevice(person.deviceName, person.fingerprint)"
        >
          <template #actions>
            <Button
              variant="ghost"
              size="small"
              :disabled="state.isBusy"
              @click="state.removeAccess(person).catch(state.fail)"
            >
              {{ words.removeAccess }}
            </Button>
          </template>
        </SyncRow>
      </ul>
    </section>
    <template #actions>
      <template v-if="isAsking">
        <Button variant="ghost" :disabled="state.isBusy" @click="onDecline">
          {{ words.decline }}
        </Button>
        <Button :disabled="state.isBusy || !isConfirmed" @click="onAllow">{{ words.approve }}</Button>
      </template>
      <template v-else>
        <template v-if="state.isLinkAlive">
          <Button variant="ghost" :disabled="state.isBusy" @click="onRevoke">
            {{ words.revoke }}
          </Button>
          <Button variant="outline" @click="onCopy">
            <Copy :size="14" aria-hidden="true" />
            {{ words.copyLinkButton }}
          </Button>
        </template>
        <Button @click="state.close">{{ words.done }}</Button>
      </template>
    </template>
  </Dialog>
</template>
