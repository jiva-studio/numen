<script setup lang="ts">
/** The new device the host is asked about: its name and fingerprint. */
import { computed } from 'vue'
import { WORDS as words } from '../../words'
import type { Fact } from '../../lib/types'
import LinkFacts from './LinkFacts.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  name: string
  fingerprint: string
}>()

/* --------------------------------- State ---------------------------------- */
const facts = computed<Fact[]>(() => [
  { label: words.linkDeviceName, value: props.name, isMono: false },
  { label: words.linkDeviceFingerprint, value: props.fingerprint, isMono: true },
])
</script>

<template>
  <div class="link-peer">
    <LinkFacts :facts="facts" />
    <p class="link-peer__ask">{{ words.linkChecking }}</p>
  </div>
</template>

<style scoped>
.link-peer {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.link-peer__ask {
  margin: 0;
}
</style>
