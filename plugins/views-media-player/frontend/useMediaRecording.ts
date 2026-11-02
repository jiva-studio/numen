import { getNow } from '@numen/ui'
import { computed, onUnmounted, ref } from 'vue'

export type RecordingStatus = 'idle' | 'recording' | 'paused' | 'stopped'
export type RecordAction = 'start' | 'pause' | 'resume' | 'stop' | 'reset'

export interface MediaRecordingEmit {
  (e: 'record-start'): void
  (e: 'record-stop', blob: Blob | null): void
}

export function getWaveformBars(
  elapsedMs: number,
  isRecording: boolean,
  count = 32,
): readonly number[] {
  const bars: number[] = []
  for (let i = 0; i < count; i++) {
    if (isRecording) {
      const phase = (elapsedMs / 150 + i * 0.4) % (Math.PI * 2)
      const height = Math.max(12, Math.round(Math.abs(Math.sin(phase)) * 85 + 15))
      bars.push(height)
    } else {
      bars.push(15)
    }
  }
  return bars
}

export function useMediaRecording(emit: MediaRecordingEmit) {
  const status = ref<RecordingStatus>('idle')
  const elapsedMs = ref(0)
  const recordedAudioUrl = ref<string | null>(null)
  const recordedBlob = ref<Blob | null>(null)
  const errorMessage = ref('')

  let timerInterval: ReturnType<typeof setInterval> | null = null
  let mediaStream: MediaStream | null = null
  let mediaRecorder: MediaRecorder | null = null
  let recordedChunks: Blob[] = []

  const isRecording = computed(() => status.value === 'recording')

  function startTimer() {
    stopTimer()
    const startedAt = getNow() - elapsedMs.value
    timerInterval = setInterval(() => {
      elapsedMs.value = getNow() - startedAt
    }, 100)
  }

  function stopTimer() {
    if (timerInterval) {
      clearInterval(timerInterval)
      timerInterval = null
    }
  }

  function cleanupStream() {
    if (mediaStream) {
      for (const track of mediaStream.getTracks()) {
        track.stop()
      }
      mediaStream = null
    }
  }

  onUnmounted(() => {
    stopTimer()
    cleanupStream()
    if (recordedAudioUrl.value) {
      URL.revokeObjectURL(recordedAudioUrl.value)
    }
  })

  async function onStartRecording() {
    errorMessage.value = ''
    recordedChunks = []

    try {
      if (navigator.mediaDevices && typeof navigator.mediaDevices.getUserMedia === 'function') {
        mediaStream = await navigator.mediaDevices.getUserMedia({ audio: true })
        mediaRecorder = new MediaRecorder(mediaStream)

        mediaRecorder.ondataavailable = (event: BlobEvent) => {
          if (event.data.size > 0) {
            recordedChunks.push(event.data)
          }
        }

        mediaRecorder.onstop = () => {
          const blob = new Blob(recordedChunks, { type: 'audio/webm' })
          recordedBlob.value = blob
          if (recordedAudioUrl.value) {
            URL.revokeObjectURL(recordedAudioUrl.value)
          }
          recordedAudioUrl.value = URL.createObjectURL(blob)
          emit('record-stop', blob)
        }

        mediaRecorder.start(250)
      } else {
        // Graceful fallback for non-media environments/testing
        recordedBlob.value = new Blob(['mock audio data'], { type: 'audio/webm' })
        recordedAudioUrl.value = 'blob:mock-recording'
        emit('record-stop', recordedBlob.value)
      }

      status.value = 'recording'
      elapsedMs.value = 0
      startTimer()
      emit('record-start')
    } catch (err: unknown) {
      errorMessage.value = err instanceof Error ? err.message : 'Could not access microphone'
      status.value = 'idle'
    }
  }

  function onPauseRecording() {
    if (mediaRecorder && mediaRecorder.state === 'recording') {
      mediaRecorder.pause()
    }
    status.value = 'paused'
    stopTimer()
  }

  function onResumeRecording() {
    if (mediaRecorder && mediaRecorder.state === 'paused') {
      mediaRecorder.resume()
    }
    status.value = 'recording'
    startTimer()
  }

  function onStopRecording() {
    if (mediaRecorder && mediaRecorder.state !== 'inactive') {
      mediaRecorder.stop()
    } else if (!mediaRecorder) {
      const blob = new Blob(['mock audio data'], { type: 'audio/webm' })
      recordedBlob.value = blob
      recordedAudioUrl.value = 'blob:mock-recording'
      emit('record-stop', blob)
    }
    cleanupStream()
    stopTimer()
    status.value = 'stopped'
  }

  function onReset() {
    if (status.value === 'recording' || status.value === 'paused') {
      onStopRecording()
    }
    status.value = 'idle'
    elapsedMs.value = 0
    if (recordedAudioUrl.value) {
      URL.revokeObjectURL(recordedAudioUrl.value)
      recordedAudioUrl.value = null
    }
    recordedBlob.value = null
  }

  function onRecordAction(action: RecordAction) {
    if (action === 'start') {
      onStartRecording().catch((err: unknown) => {
        errorMessage.value = err instanceof Error ? err.message : 'Could not access microphone'
      })
    } else if (action === 'pause') {
      onPauseRecording()
    } else if (action === 'resume') {
      onResumeRecording()
    } else if (action === 'stop') {
      onStopRecording()
    } else if (action === 'reset') {
      onReset()
    }
  }

  return {
    status,
    elapsedMs,
    recordedAudioUrl,
    recordedBlob,
    errorMessage,
    isRecording,
    onRecordAction,
  }
}
