import { asFailure, asValue, type Result, formatErrorMessage } from '@numen/plugin-sdk'
export const BAR_COUNT = 32

export interface Microphone {
  readLevels(): number[]
  pause(): void
  resume(): void
  stop(): void
  cancel(): void
}

interface CaptureEvents {
  complete(audio: Blob): Promise<void>
  fail(message: string): void
}

export async function openMicrophone(events: CaptureEvents): Promise<Result<Microphone>> {
  if (!navigator.mediaDevices?.getUserMedia || typeof MediaRecorder === 'undefined') {
    return asFailure('Microphone recording is unavailable in this window')
  }
  let stream: MediaStream | undefined
  let context: AudioContext | undefined
  const release = () => {
    stream?.getTracks().forEach((track) => track.stop())
    context?.close().catch((error: unknown) => events.fail(formatErrorMessage(error)))
  }
  try {
    const mimeType = ['audio/mp4', 'audio/ogg;codecs=opus', 'audio/webm;codecs=opus'].find((type) =>
      MediaRecorder.isTypeSupported(type),
    )
    if (!mimeType) return asFailure('No supported audio recording format is available')
    stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    const recorder = new MediaRecorder(stream, { mimeType })
    context = new AudioContext()
    const analyser = context.createAnalyser()
    analyser.fftSize = BAR_COUNT * 2
    context.createMediaStreamSource(stream).connect(analyser)
    const samples = new Uint8Array(BAR_COUNT)
    const chunks: Blob[] = []
    recorder.ondataavailable = ({ data }) => {
      if (data.size) chunks.push(data)
    }
    recorder.onerror = () => {
      recorder.onstop = null
      release()
      events.fail('Microphone recording failed')
    }
    recorder.onstop = async () => {
      release()
      if (!chunks.length) return events.fail('The microphone captured no audio')
      await events.complete(new Blob(chunks, { type: recorder.mimeType || mimeType }))
    }
    recorder.start(250)
    return asValue({
      readLevels() {
        analyser.getByteFrequencyData(samples)
        return Array.from(samples, (level) => (level * 100) / 255)
      },
      pause: () => recorder.pause(),
      resume: () => recorder.resume(),
      stop: () => recorder.stop(),
      cancel() {
        recorder.onstop = null
        if (recorder.state !== 'inactive') recorder.stop()
        release()
      },
    })
  } catch (error) {
    release()
    return asFailure(formatErrorMessage(error))
  }
}
