<script setup lang="ts">
/** The new device the host is asked about: its name and fingerprint, and the code to read out when this device scanned it. */
import { computed } from 'vue'
import { Button } from '@numen/ui'
import { WORDS as words } from '../../words'
import type { Fact } from '../../lib/types'
import LinkFacts from './LinkFacts.vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  name: string
  fingerprint: string
  code: string
  isApproved: boolean
  isBusy: boolean
}>()

const emit = defineEmits<{ (e: 'decide', isApproved: boolean): void }>()

/* --------------------------------- State ---------------------------------- */
const facts = computed<Fact[]>(() => [
  { label: words.linkDeviceName, value: props.name, isMono: false },
  { label: words.linkDeviceFingerprint, value: props.fingerprint, isMono: true },
  ...(props.code === '' ? [] : [{ label: words.linkCode, value: props.code, isMono: true }]),
])
</script>

<template>
  <div class="link-peer">
    <LinkFacts :facts="facts" />
    <p>{{ words.linkCompare }}</p>
    <p v-if="isApproved">{{ words.linkWaitingCode }}</p>
    <div v-else class="link-peer__actions">
      <Button type="button" variant="solid" :disabled="isBusy" @click="emit('decide', true)">
        {{ words.linkApprove }}
      </Button>
      <Button type="button" variant="outline" :disabled="isBusy" @click="emit('decide', false)">
        {{ words.linkDecline }}
      </Button>
    </div>
  </div>
</template>

<style scoped>
.link-peer {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.link-peer__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}
</style>
