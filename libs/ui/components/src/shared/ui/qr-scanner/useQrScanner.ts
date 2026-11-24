import { onBeforeUnmount, onMounted, ref, type ShallowRef } from 'vue'
import { describeCameraFailure, FRAMES_PER_SECOND, type ScannerCamera } from '@/shared/api/camera'
import { doc } from '@/shared/api/dom'
import { decodeFrame } from './decodeFrame'

/** How long a code that stays in front of the camera is not reported again. */
const SAME_CODE_PAUSE_MS = 2000

export interface QrScannerSource {
  camera: ScannerCamera
  isAccepted: (text: string) => boolean
  continuous: boolean
}

export interface QrScannerEvents {
  (e: 'scanned', text: string): void
  (e: 'failed', reason: string): void
  (e: 'unavailable', reason: string): void
}

/**
 * The camera of one scanner and the frames read from it: what it is waiting for, the problem it shows, and a way to
 * ask the camera again after a failure. The stream stops after the first scan; in the continuous mode it stops only
 * when the scanner unmounts.
 */
export function useQrScanner(
  video: Readonly<ShallowRef<HTMLVideoElement | null>>,
  source: QrScannerSource,
  emit: QrScannerEvents,
) {
  const status = ref<'waiting' | 'scanning' | 'failed'>('waiting')
  const problem = ref('')
  let stream: MediaStream | undefined
  let timer: ReturnType<typeof setInterval> | undefined
  let rejected = ''
  let lastText = ''
  let lastAt = 0

  function readFrame(): void {
    const view = video.value
    if (view === null || view.videoWidth === 0) return
    const canvas = doc.createElement('canvas')
    canvas.width = view.videoWidth
    canvas.height = view.videoHeight
    const drawing = canvas.getContext('2d', { willReadFrequently: true })
    if (drawing === null) return
    drawing.drawImage(view, 0, 0)
    const pixels = drawing.getImageData(0, 0, canvas.width, canvas.height)
    const found = decodeFrame(pixels)
    if (found === null) return
    if (!source.isAccepted(found)) {
      if (found === rejected) return
      rejected = found
      problem.value = 'This code is not a link of Numen.'
      emit('failed', problem.value)
      return
    }
    if (source.continuous) {
      const now = Date.now()
      const isRepeat = found === lastText && now - lastAt < SAME_CODE_PAUSE_MS
      lastText = found
      lastAt = now
      if (isRepeat) return
      problem.value = ''
      emit('scanned', found)
      return
    }
    stop()
    emit('scanned', found)
  }

  function stop(): void {
    if (timer !== undefined) clearInterval(timer)
    timer = undefined
    stream?.getTracks().forEach((track) => track.stop())
    stream = undefined
  }

  async function start(): Promise<void> {
    status.value = 'waiting'
    problem.value = ''
    try {
      stream = await source.camera()
    } catch (cause) {
      status.value = 'failed'
      problem.value = describeCameraFailure(cause)
      emit('unavailable', problem.value)
      return
    }
    if (video.value === null) return stop()
    video.value.srcObject = stream
    await Promise.resolve(video.value.play()).catch(() => undefined)
    status.value = 'scanning'
    timer = setInterval(readFrame, 1000 / FRAMES_PER_SECOND)
  }

  onMounted(start)
  onBeforeUnmount(stop)

  return { status, problem, retry: start }
}
