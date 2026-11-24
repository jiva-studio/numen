<script setup lang="ts">
/** What a link in progress draws in each view: a status, the QR code, the new device, the code, the account, the end. */
import { computed } from 'vue'
import { ErrorState } from '@numen/ui'
import type { LinkInfo, LinkRole } from '@/entities/identity'
import type { Fact } from '../../lib/types'
import type { LinkView } from '../../lib/phases'
import { WORDS as words } from '../../words'
import LinkCodeForm from './LinkCodeForm.vue'
import LinkFacts from './LinkFacts.vue'
import LinkPeerView from './LinkPeerView.vue'
import LinkQrView from './LinkQrView.vue'
import LinkStatus from './LinkStatus.vue'
import LinkSteps from './LinkSteps.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  view: LinkView
  link: LinkInfo | undefined
  role: LinkRole
  problem: string
  steps: readonly string[]
  code: string
}>()

const emit = defineEmits<{ (e: 'update:code', value: string): void }>()

/* --------------------------------- State ---------------------------------- */
const ownFacts = computed<Fact[]>(() =>
  props.role === 'joiner' && props.link !== undefined
    ? [
        { label: words.linkDeviceName, value: props.link.deviceName, isMono: false },
        { label: words.linkDeviceFingerprint, value: props.link.deviceFingerprint, isMono: true },
      ]
    : [],
)
const isOwnShown = computed(() => ['qr', 'waiting', 'code', 'account'].includes(props.view))
</script>

<template>
  <div class="link-progress">
    <LinkSteps v-if="view === 'qr' || view === 'preparing'" :steps="steps" />
    <LinkFacts v-if="isOwnShown && ownFacts.length > 0" :facts="ownFacts" />

    <LinkStatus
      v-if="view === 'preparing'"
      :title="words.linkPreparingTitle"
      :detail="words.linkPreparingDetail"
    />
    <ErrorState
      v-else-if="view === 'stalled'"
      :heading="words.linkStalledTitle"
      :message="words.linkStalled"
    />
    <ErrorState
      v-else-if="view === 'refused'"
      :heading="words.linkRefusedTitle"
      :message="problem"
    />
    <LinkQrView
      v-else-if="view === 'qr' && link"
      :qr-text="link.qrText"
      :account-fingerprint="role === 'host' ? link.accountFingerprint : ''"
    />
    <LinkStatus
      v-else-if="view === 'waiting'"
      :title="words.linkWaitingTitle"
      :detail="words.linkWaitingDetail"
    />
    <LinkPeerView
      v-else-if="view === 'peer' && link"
      :name="link.peerName"
      :fingerprint="link.peerFingerprint"
      :code="link.code"
      :is-approved="link.isApproved"
    />
    <LinkCodeForm
      v-else-if="view === 'code' && link"
      :model-value="code"
      :tries-left="link.triesLeft"
      @update:model-value="emit('update:code', $event)"
    />
    <p v-else-if="view === 'account' && link">
      {{ words.linkJoinQuestion(link.accountFingerprint) }}
    </p>
    <LinkStatus
      v-else-if="view === 'linked'"
      is-done
      :title="words.linkDoneTitle"
      :detail="words.linkDoneDetail"
    />
    <ErrorState
      v-else-if="view === 'failed'"
      :heading="words.linkFailedTitle"
      :message="words.linkReason(link?.reason)"
    />
    <LinkStatus v-else :title="words.linkLinkingTitle" :detail="words.linkLinkingDetail" />
    <p v-if="problem !== '' && view !== 'refused'" role="alert" class="link-progress__problem">
      {{ problem }}
    </p>
  </div>
</template>

<style scoped>
.link-progress {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.link-progress__problem {
  margin: 0;
  color: var(--numen-alarm);
}
</style>
