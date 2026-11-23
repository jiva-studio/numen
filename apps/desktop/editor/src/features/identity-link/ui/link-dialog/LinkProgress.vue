<script setup lang="ts">
/** What a link in progress shows in each of its phases. */
import { computed } from 'vue'
import type { LinkInfo, LinkRole } from '@/entities/identity'
import { WORDS as words } from '../../words'
import LinkCodeForm from './LinkCodeForm.vue'
import type { Fact } from '../../lib/types'
import LinkFacts from './LinkFacts.vue'
import LinkJoinQuestion from './LinkJoinQuestion.vue'
import LinkPeerView from './LinkPeerView.vue'
import LinkQrView from './LinkQrView.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ link: LinkInfo; role: LinkRole; isBusy: boolean }>()

const emit = defineEmits<{
  (e: 'decide', isApproved: boolean): void
  (e: 'code', code: string): void
  (e: 'join', isJoining: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const ownFacts = computed<Fact[]>(() =>
  props.role === 'joiner'
    ? [
        { label: words.linkDeviceName, value: props.link.deviceName, isMono: false },
        { label: words.linkDeviceFingerprint, value: props.link.deviceFingerprint, isMono: true },
      ]
    : [],
)
const isWaiting = computed(() =>
  ['waiting_for_peer', 'waiting_for_accept'].includes(props.link.phase),
)
const isShowingQr = computed(
  () => props.link.phase === 'waiting_for_peer' && props.link.mode === 'show',
)
const hostAccount = computed(() => (props.role === 'host' ? props.link.accountFingerprint : ''))
</script>

<template>
  <div class="link-progress">
    <LinkFacts v-if="ownFacts.length > 0" :facts="ownFacts" />
    <p v-if="link.phase === 'waiting_for_channel'">{{ words.linkPreparing }}</p>
    <LinkQrView v-else-if="isShowingQr" :qr-text="link.qrText" :account-fingerprint="hostAccount" />
    <p v-else-if="isWaiting">{{ words.linkWaiting }}</p>
    <LinkPeerView
      v-else-if="link.phase === 'confirm_peer'"
      :name="link.peerName"
      :fingerprint="link.peerFingerprint"
      :code="link.code"
      :is-approved="link.isApproved"
      :is-busy="isBusy"
      @decide="emit('decide', $event)"
    />
    <LinkCodeForm
      v-else-if="link.phase === 'enter_code'"
      :tries-left="link.triesLeft"
      :is-busy="isBusy"
      @submit="emit('code', $event)"
    />
    <LinkJoinQuestion
      v-else-if="link.phase === 'confirm_account' && role === 'joiner'"
      :account="link.accountFingerprint"
      :is-busy="isBusy"
      @answer="emit('join', $event)"
    />
    <p v-else-if="link.phase === 'linked'">{{ words.linkDone }}</p>
    <p v-else-if="link.phase === 'failed'" role="alert">{{ words.linkReason(link.reason) }}</p>
    <p v-else>{{ words.linkLinking }}</p>
  </div>
</template>

<style scoped>
.link-progress {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
</style>
