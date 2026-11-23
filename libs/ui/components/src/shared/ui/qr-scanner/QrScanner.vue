
<script setup lang="ts">
/**
 * Reads QR codes from the camera at eight frames a second and reports the text of the first valid one it finds.
 * The stream stops when the scanner unmounts. The decoder is plain JavaScript, so no worker and no wasm is needed.
 */
import jsQR from 'jsqr'
import { onBeforeUnmount, onMounted, ref } from 'vue'
import {
  describeCameraFailure,
  FRAMES_PER_SECOND,
  openMachineCamera,
  type ScannerCamera,
} from '@/shared/lib/camera/camera'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(defineProps<{ camera?: ScannerCamera; accepts?: (text: string) => boolean }>(), {
  camera: openMachineCamera,
  accepts: () => true,
})

const emit = defineEmits<{
  (e: 'scanned', text: string): void
  (e: 'failed', reason: string): void
}>()

/* --------------------------------- State ---------------------------------- */
const video = ref<HTMLVideoElement | null>(null)
const status = ref<'waiting' | 'scanning' | 'failed'>('waiting')
const problem = ref('')
let stream: MediaStream | undefined
let timer: ReturnType<typeof setInterval> | undefined
let rejected = ''

/* -------------------------------- Handlers -------------------------------- */
function readFrame(): void {
  const view = video.value
  if (view === null || view.videoWidth === 0) return
  const canvas = document.createElement('canvas')
  canvas.width = view.videoWidth
  canvas.height = view.videoHeight
  const drawing = canvas.getContext('2d', { willReadFrequently: true })
  if (drawing === null) return
  drawing.drawImage(view, 0, 0)
  const pixels = drawing.getImageData(0, 0, canvas.width, canvas.height)
  const found = jsQR(pixels.data, pixels.width, pixels.height)
  if (found === null) return
  if (!props.accepts(found.data)) {
    if (found.data === rejected) return
    rejected = found.data
    problem.value = 'This code is not a link of Numen.'
    emit('failed', problem.value)
    return
  }
  stop()
  emit('scanned', found.data)
}

function stop(): void {
  if (timer !== undefined) clearInterval(timer)
  timer = undefined
  stream?.getTracks().forEach((track) => track.stop())
  stream = undefined
}

async function start(): Promise<void> {
  try {
    stream = await props.camera()
  } catch (cause) {
    status.value = 'failed'
    problem.value = describeCameraFailure(cause)
    emit('failed', problem.value)
    return
  }
  if (video.value === null) return stop()
  video.value.srcObject = stream
  await video.value.play().catch(() => undefined)
  status.value = 'scanning'
  timer = setInterval(readFrame, 1000 / FRAMES_PER_SECOND)
}

onMounted(start)
onBeforeUnmount(stop)
</script>

<template>
  <div class="qr-scanner">
    <video ref="video" class="qr-scanner__video" muted playsinline />
    <p v-if="status === 'waiting'" class="qr-scanner__note">Waiting for the camera.</p>
    <p v-if="problem !== ''" role="alert" class="qr-scanner__note">{{ problem }}</p>
  </div>
</template>

<style scoped>
.qr-scanner {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  align-items: center;
}

.qr-scanner__video {
  width: 100%;
  max-width: 22rem;
  border-radius: 0.5rem;
  background: var(--numen-surface-sunken, #000);
}

.qr-scanner__note {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}
</style>
