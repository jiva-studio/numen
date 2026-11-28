<script setup lang="ts">
/**
 * Reads QR codes from the camera at eight frames a second and reports the text of the first valid one it finds.
 * It asks the machine for the camera when it mounts, so a dialog that holds it asks once it is open. One box of 4:3 fills the width it
 * is given from the first frame: a spinner shows in it while the camera starts, the picture replaces it, and a failure
 * is written in it. The stream stops after the first scan; with `isContinuous` it
 * keeps reading and reports each new code (the same code held in front of the camera once) until the scanner unmounts.
 * The decoder is plain JavaScript, so no worker and no wasm is needed.
 */
import { useTemplateRef } from 'vue'
import { openMachineCamera, type ScannerCamera } from '@/shared/api/camera'
import { Spinner } from '../spinner'
import { useQrScanner } from './useQrScanner'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(
  defineProps<{
    camera?: ScannerCamera
    isAccepted?: (text: string) => boolean
    isContinuous?: boolean
    rejectedMessage?: string
  }>(),
  {
    camera: openMachineCamera,
    isAccepted: () => true,
    isContinuous: false,
    rejectedMessage: 'This code is not accepted.',
  },
)

const emit = defineEmits<{
  (e: 'scanned', text: string): void
  (e: 'failed', reason: string): void
  (e: 'unavailable', reason: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const video = useTemplateRef<HTMLVideoElement>('video')
const { status, problem } = useQrScanner(video, props, emit)
</script>

<template>
  <div class="qr-scanner">
    <video ref="video" class="qr-scanner__video" autoplay muted playsinline />
    <div v-if="status === 'waiting'" class="qr-scanner__cover" role="status" aria-label="Starting the camera">
      <Spinner />
    </div>
    <p v-if="problem !== ''" role="alert" class="qr-scanner__cover qr-scanner__problem">{{ problem }}</p>
  </div>
</template>

<style scoped>
.qr-scanner {
  position: relative;
  inline-size: 100%;
  aspect-ratio: 4 / 3;
  overflow: hidden;
  border-radius: var(--numen-radius);
  background: var(--numen-field-bg);
}

.qr-scanner__video {
  display: block;
  inline-size: 100%;
  block-size: 100%;
  object-fit: cover;
}

.qr-scanner__cover {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0;
  color: var(--numen-hushed);
}

.qr-scanner__problem {
  padding: 1rem;
  color: var(--numen-ink);
  background: color-mix(in srgb, var(--numen-surface) 88%, transparent);
  text-align: center;
  font-size: var(--numen-text-2);
}
</style>
