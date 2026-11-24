<script setup lang="ts">
/** The new device the host is asked about: its name and fingerprint, and the confirmation code to type on it. */
import { computed } from 'vue'
import { WORDS as words } from '../../words'
import type { Fact } from '../../lib/types'
import LinkFacts from './LinkFacts.vue'
import LinkStatus from './LinkStatus.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  name: string
  fingerprint: string
  code: string
  isApproved: boolean
}>()

/* --------------------------------- State ---------------------------------- */
const facts = computed<Fact[]>(() => [
  { label: words.linkDeviceName, value: props.name, isMono: false },
  { label: words.linkDeviceFingerprint, value: props.fingerprint, isMono: true },
])
const grouped = computed(() => props.code.replace(/^(\d{3})(\d+)$/, '$1 $2'))
</script>

<template>
  <div class="link-peer">
    <LinkFacts :facts="facts" />
    <p v-if="!isApproved" class="link-peer__ask">{{ words.linkChecking }}</p>
    <LinkStatus
      v-if="isApproved"
      :title="words.linkWaitingCodeTitle"
      :detail="words.linkWaitingCodeDetail"
    >
      <p v-if="code !== ''" class="link-peer__code" :aria-label="words.linkCode">{{ grouped }}</p>
    </LinkStatus>
    <div v-else-if="code !== ''" class="link-peer__shown">
      <p class="link-peer__hint">{{ words.linkCodeHint }}</p>
      <p class="link-peer__code" :aria-label="words.linkCode">{{ grouped }}</p>
    </div>
  </div>
</template>

<style scoped>
.link-peer {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.link-peer__ask,
.link-peer__hint {
  margin: 0;
}

.link-peer__shown {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.25rem;
}

.link-peer__hint {
  color: var(--numen-hushed);
}

.link-peer__code {
  margin: 0;
  color: var(--numen-ink);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-4);
  font-weight: 600;
  letter-spacing: 0.1em;
}
</style>
