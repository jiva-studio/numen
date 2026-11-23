<script setup lang="ts">
/**
 * The page "Shares" of the plugin's settings: the code this account gives to a person who shares a folder,
 * the field where an invitation is received, and the folders received.
 */
import { computed, onMounted } from 'vue'
import { Button } from '@numen/ui'
import type { SharesClient } from '../api/shares'
import { createReceivedView } from '../model/shares'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  client?: SharesClient
  refreshMs?: number
}>()

/* --------------------------------- State ---------------------------------- */
const view = createReceivedView({ client: props.client, refreshMs: props.refreshMs })
const problem = computed(() => view.error || view.invitationError)
const canReceive = computed(() => !view.isBusy && view.text.trim() !== '')

/* --------------------------------- Hooks ---------------------------------- */
onMounted(() => {
  report(view.load())
})

/* -------------------------------- Handlers -------------------------------- */
function onInvitationInput(event: Event): void {
  report(view.setText((event.target as HTMLTextAreaElement).value))
}

function onCopy(): void {
  report(navigator.clipboard.writeText(view.code))
}

/* --------------------------------- Helpers -------------------------------- */
function report(work: Promise<unknown>): void {
  work.catch((failure: unknown) => {
    view.fail(failure)
  })
}
</script>

<template>
  <section class="shares" :aria-label="words.sharesTitle">
    <h2 class="shares__title">{{ words.sharesTitle }}</h2>
    <div class="shares__block">
      <input class="shares__input" readonly :value="view.code" :aria-label="words.yourCode" />
      <Button variant="outline" @click="onCopy">{{ words.copy }}</Button>
      <p class="shares__note">{{ words.yourCodeHint }}</p>
      <p v-if="view.fingerprint !== ''" class="shares__note">
        {{ words.yourFingerprint }}: <output>{{ view.fingerprint }}</output>
      </p>
    </div>
    <div class="shares__block">
      <textarea
        class="shares__input shares__input--invitation"
        :value="view.text"
        :aria-label="words.invitation"
        :placeholder="words.pasteInvitation"
        @input="onInvitationInput"
      />
      <p v-if="view.shareName !== ''" class="shares__note">
        <span class="shares__name">{{ view.shareName }}</span>
        <span>{{ words.ownerFingerprint }}:</span>
        <output>{{ view.ownerFingerprint }}</output>
      </p>
      <Button variant="outline" :disabled="!canReceive" @click="report(view.receive())">
        {{ words.receive }}
      </Button>
    </div>
    <p v-if="problem !== ''" class="shares__note shares__note--alarm" role="alert">{{ problem }}</p>
    <p v-if="view.rows.length === 0" class="shares__note">{{ words.noReceived }}</p>
    <ul v-else class="shares__list" :aria-label="words.received">
      <li v-for="row in view.rows" :key="row.vaultId" class="shares__row">
        <span class="shares__name">{{ row.name }}</span>
        <span class="shares__note">{{ row.stateLabel }}</span>
      </li>
    </ul>
  </section>
</template>
