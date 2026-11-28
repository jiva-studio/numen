import { formatErrorMessage } from '@numen/plugin-sdk'
import { getCurrentDate, getNow } from '@numen/ui'
import { computed, onUnmounted, ref } from 'vue'
import { BAR_COUNT, openMicrophone, type Microphone } from '../api/microphone'
import type { RecorderTab } from '../api/saveRecording'

export type RecordingStatus = 'idle' | 'starting' | 'recording' | 'paused' | 'saving' | 'stopped'
export type RecordAction = 'start' | 'pause' | 'resume' | 'stop' | 'reset'

function getErrorMessage(error: unknown): string {
  return formatErrorMessage(error)
}

export function useMediaRecording(getTab: () => RecorderTab | undefined) {
  const status = ref<RecordingStatus>('idle')
  const elapsedMs = ref(0)
  const bars = ref<number[]>(Array<number>(BAR_COUNT).fill(0))
  const recordedAudioUrl = ref('')
  const errorMessage = ref('')
  const isRecording = computed(() => status.value === 'recording')
  let isClosed = false
  let interval: ReturnType<typeof setInterval> | undefined
  let microphone: Microphone | undefined
  let startedAt = 0
  let captureDate = ''
  let audio: Blob | undefined
  let savedPath = ''
  let target: RecorderTab | undefined

  function stopMeter() {
    clearInterval(interval)
    bars.value = Array<number>(BAR_COUNT).fill(0)
  }

  function startMeter() {
    startedAt = getNow() - elapsedMs.value
    interval = setInterval(() => {
      elapsedMs.value = getNow() - startedAt
      if (microphone) bars.value = microphone.readLevels()
    }, 100)
  }

  function reset() {
    if (recordedAudioUrl.value) URL.revokeObjectURL(recordedAudioUrl.value)
    recordedAudioUrl.value = ''
    elapsedMs.value = 0
    audio = undefined
    savedPath = ''
    errorMessage.value = ''
    status.value = 'idle'
  }

  async function save() {
    if (!audio || status.value === 'saving') return
    status.value = 'saving'
    errorMessage.value = ''
    try {
      const tab = target
      if (!tab) throw new Error('The recording tab has no vault')
      if (!savedPath) {
        const result = await tab.save(audio, captureDate)
        if (!result.ok) throw new Error(result.error)
        savedPath = result.value
      }
      if (!isClosed) await tab.open(savedPath)
    } catch (error) {
      errorMessage.value = getErrorMessage(error)
    } finally {
      status.value = 'stopped'
    }
  }

  async function start() {
    if (status.value !== 'idle') return
    target = getTab()
    captureDate = getCurrentDate().toISOString()
    status.value = 'starting'
    errorMessage.value = ''
    const result = await openMicrophone({
      async complete(blob) {
        microphone = undefined
        audio = blob
        stopMeter()
        if (!isClosed) recordedAudioUrl.value = URL.createObjectURL(audio)
        status.value = 'stopped'
        await save()
      },
      fail(message) {
        microphone = undefined
        stopMeter()
        errorMessage.value = message
        status.value = 'idle'
      },
    })
    if (!result.ok) {
      errorMessage.value = result.error
      status.value = 'idle'
      return
    }
    if (isClosed) {
      result.value.cancel()
      return
    }
    microphone = result.value
    status.value = 'recording'
    startMeter()
  }

  function pause() {
    if (microphone && isRecording.value) {
      microphone.pause()
      elapsedMs.value = getNow() - startedAt
      stopMeter()
      status.value = 'paused'
    }
  }

  function resume() {
    if (microphone && status.value === 'paused') {
      microphone.resume()
      status.value = 'recording'
      startMeter()
    }
  }

  function stop() {
    if (microphone) {
      if (isRecording.value) elapsedMs.value = getNow() - startedAt
      stopMeter()
      status.value = 'saving'
      microphone.stop()
    }
  }

  async function onRecordAction(action: RecordAction) {
    const actions = { start, reset, pause, resume, stop }
    await actions[action]()
  }

  onUnmounted(() => {
    isClosed = true
    if (status.value !== 'saving') microphone?.cancel()
    stopMeter()
    if (recordedAudioUrl.value) URL.revokeObjectURL(recordedAudioUrl.value)
  })

  return {
    status,
    elapsedMs,
    bars,
    recordedAudioUrl,
    errorMessage,
    isRecording,
    onRecordAction,
    save,
  }
}
