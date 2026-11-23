<script lang="ts">
/** Opens the camera; the scanner asks the machine for one unless a caller gives its own. */
export type ScannerCamera = () => Promise<MediaStream>

const FRAMES_PER_SECOND = 8

const openMachineCamera: ScannerCamera = () =>
  navigator.mediaDevices.getUserMedia({ video: { facingMode: 'environment' } })

/** Whether this machine can open a camera at all. */
export const hasCamera = (): boolean => typeof navigator.mediaDevices?.getUserMedia === 'function'

/** What a person reads when the camera does not open. */
export const describeCameraFailure = (cause: unknown): string => {
  const name = cause instanceof DOMException ? cause.name : ''
  if (name === 'NotAllowedError') return 'The camera is not allowed.'
  if (name === 'NotFoundError') return 'No camera was found.'
  return 'The camera did not open.'
}

export { FRAMES_PER_SECOND, openMachineCamera }
</script>

<script setup lang="ts">
/**
 * Reads QR codes from the camera at eight frames a second and says the text of the first valid one it sees.
 * The stream stops when the scanner goes away. The decoder is plain JavaScript, so no worker and no wasm is needed.
 */
import jsQR from 'jsqr'
import { onBeforeUnmount, onMounted, ref } from 'vue'

/* ----------------------------- Props & Emits ------------------------------ */
const props = withDefaults(defineProps<{ camera?: ScannerCamera; isLink?: (text: string) => boolean }>(), {
  camera: openMachineCamera,
  isLink: () => true,
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
  if (!props.isLink(found.data)) {
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
