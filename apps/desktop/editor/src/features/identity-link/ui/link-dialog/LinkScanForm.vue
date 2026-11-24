<script setup lang="ts">
/**
 * What a new device scans with: the camera in a box of the size it keeps while the camera is checked, and, once
 * the person asks for it or the machine has no camera, a field to paste the link of the other device into.
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
  isPasting: boolean
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
    <div v-if="phase !== 'paste'" class="link-scan__stage">
      <LinkStatus
        v-if="phase === 'checking'"
        :title="words.linkCheckingCamera"
        :detail="words.linkCheckingCameraDetail"
      />
      <QrScanner
        v-else
        v-bind="camera === undefined ? {} : { camera }"
        @scanned="emit('scanned', $event)"
        @unavailable="emit('unavailable', $event)"
      />
    </div>
    <p v-else role="status" class="link-scan__notice">{{ notice }}</p>
    <LinkField
      v-if="isPasting || phase === 'paste'"
      :model-value="modelValue"
      :label="words.linkPaste"
      is-mono
      @update:model-value="emit('update:modelValue', $event)"
    />
    <p v-if="problem !== ''" role="alert" class="link-scan__problem">{{ problem }}</p>
  </div>
</template>

<style scoped>
.link-scan {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.link-scan__stage {
  display: flex;
  align-items: center;
  justify-content: center;
  inline-size: 100%;
  max-inline-size: 22rem;
  margin-inline: auto;
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
