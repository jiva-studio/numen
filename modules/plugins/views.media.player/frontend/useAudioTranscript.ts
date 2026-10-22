import { computed, ref, type Ref } from 'vue'
import { readState, type ExternalPlayerState } from './useAudioPlayback'
import type { CueItem } from './AudioTranscript.vue'

export function useAudioTranscript(
  props: { readonly state?: ExternalPlayerState },
  currentPosition: Ref<number>,
  totalDuration: Ref<number>,
) {
  const internalCues = ref<readonly CueItem[]>([])
  const isTranscriptDeleted = ref(false)

  const transcriptCues = computed<readonly CueItem[]>(() => {
    if (isTranscriptDeleted.value) return []
    if (internalCues.value.length > 0) return internalCues.value
    const cues = readState<readonly CueItem[]>(props.state?.cues)
    if (Array.isArray(cues)) return cues
    const transcript = readState<{ cues?: readonly CueItem[] }>(props.state?.transcript)
    if (transcript?.cues) return transcript.cues
    const prose = readState<string>(props.state?.prose)
    if (prose) {
      return [{ text: prose, from: 0, to: totalDuration.value || 1000 }]
    }
    return []
  })

  const isTranscribable = computed(() => {
    if (transcriptCues.value.length > 0) return false
    if (props.state?.transcribable !== undefined) {
      return Boolean(readState(props.state.transcribable))
    }
    return true
  })

  const isDeletable = computed(() => {
    if (props.state?.deletable !== undefined) {
      return Boolean(readState(props.state.deletable))
    }
    return true
  })

  const activeCueIndex = computed(() => {
    const now = currentPosition.value
    return transcriptCues.value.findIndex((cue) => now >= cue.from && now < cue.to)
  })

  function onTranscribe() {
    isTranscriptDeleted.value = false
    if (typeof props.state?.transcribe === 'function') {
      props.state.transcribe()
      return
    }
    if (typeof props.state?.proofread === 'function') {
      props.state.proofread()
    }
    internalCues.value = [
      { text: 'Sample audio transcription text.', from: 0, to: totalDuration.value || 1000 },
    ]
  }

  function onDeleteTranscript() {
    isTranscriptDeleted.value = true
    if (props.state?.deleteTranscript) {
      props.state.deleteTranscript()
    }
    internalCues.value = []
  }

  return {
    transcriptCues,
    isTranscribable,
    isDeletable,
    activeCueIndex,
    onTranscribe,
    onDeleteTranscript,
  }
}
