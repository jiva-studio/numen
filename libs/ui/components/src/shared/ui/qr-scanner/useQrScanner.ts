import jsQR from 'jsqr'
import { onBeforeUnmount, onMounted, ref, type ShallowRef } from 'vue'
import { describeCameraFailure, FRAMES_PER_SECOND, type ScannerCamera } from '@/shared/api/camera'
import { doc } from '@/shared/api/dom'

export interface QrScannerSource {
  camera: ScannerCamera
  isAccepted: (text: string) => boolean
}

export interface QrScannerEvents {
  (e: 'scanned', text: string): void
  (e: 'failed', reason: string): void
}

/**
 * The camera of one scanner and the frames read from it: what it is waiting for, the problem it shows, and a way to
 * ask the camera again after a failure. The stream stops when the scanner unmounts.
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
    const found = jsQR(pixels.data, pixels.width, pixels.height)
    if (found === null) return
    if (!source.isAccepted(found.data)) {
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
    status.value = 'waiting'
    problem.value = ''
    try {
      stream = await source.camera()
    } catch (cause) {
      status.value = 'failed'
      problem.value = describeCameraFailure(cause)
      emit('failed', problem.value)
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
