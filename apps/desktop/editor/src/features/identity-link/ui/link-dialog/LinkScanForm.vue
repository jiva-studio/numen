<script setup lang="ts">
/**
 * What a new device scans with: the camera across the whole width, or, only where the machine has no camera or the
 * camera failed, the reason and a field to paste the link of the other device into.
 */
import { QrScanner, type ScannerCamera } from '@numen/ui'
import { WORDS as words } from '../../words'
import LinkField from './LinkField.vue'
import LinkStatus from './LinkStatus.vue'

/* ----------------------------- Props & Emits ------------------------------ */
defineProps<{
  modelValue: string
  problem: string
  phase: 'checking' | 'camera' | 'paste'
  notice: string
  camera?: ScannerCamera
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', text: string): void
  (e: 'scanned', text: string): void
  (e: 'unavailable', reason: string): void
}>()
</script>

<template>
  <div class="link-scan">
    <div v-if="phase === 'checking'" class="link-scan__checking">
      <LinkStatus :title="words.linkCheckingCamera" :detail="words.linkCheckingCameraDetail" />
    </div>
    <QrScanner
      v-else-if="phase === 'camera'"
      v-bind="camera === undefined ? {} : { camera }"
      @scanned="emit('scanned', $event)"
      @unavailable="emit('unavailable', $event)"
    />
    <template v-else>
      <p role="status" class="link-scan__notice">{{ notice }}</p>
      <LinkField
        :model-value="modelValue"
        :label="words.linkPaste"
        is-mono
        @update:model-value="emit('update:modelValue', $event)"
      />
    </template>
    <p v-if="problem !== ''" role="alert" class="link-scan__problem">{{ problem }}</p>
  </div>
</template>

<style scoped>
.link-scan {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.link-scan__checking {
  display: flex;
  align-items: center;
  justify-content: center;
  aspect-ratio: 4 / 3;
}

.link-scan__notice {
  margin: 0;
  color: var(--numen-hushed);
}

.link-scan__problem {
  margin: 0;
  color: var(--numen-alarm);
}
</style>
