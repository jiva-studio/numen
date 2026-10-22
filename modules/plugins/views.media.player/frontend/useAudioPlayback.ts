import { computed, onMounted, onUnmounted, ref, watch, type Ref } from 'vue'

export interface ExternalPlayerState {
  readonly [key: string]: unknown
  play?(): void
  pause?(): void
  go?(ms: number): void
  seek?(ms: number): void
  setPlaybackRate?(rate: number): void
  transcribe?(): void
  proofread?(): void
  deleteTranscript?(): void
}

export interface PlaybackOptions {
  readonly path?: string
  readonly file?: string
  readonly state?: ExternalPlayerState
}

export interface PlaybackEmit {
  (e: 'play'): void
  (e: 'pause'): void
  (e: 'seek', ms: number): void
}

export function readState<T>(value: unknown): T | undefined {
  if (value && typeof value === 'object' && 'value' in value) {
    return (value as { value: T }).value
  }
  return value as T | undefined
}

export function formatClock(ms: number): string {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000))
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  return `${minutes}:${seconds.toString().padStart(2, '0')}`
}

export function useAudioPlayback(
  props: PlaybackOptions,
  emit: PlaybackEmit,
  audioElement: Ref<HTMLAudioElement | null>,
) {
  const isLocalPlaying = ref(false)
  const localPosition = ref(0)
  const localDuration = ref(0)
  const playbackRate = ref(1.0)
  const isMuted = ref(false)

  const audioSource = computed(() => {
    const url = readState<string>(props.state?.url)
    if (typeof url === 'string' && url) return url
    const mediaUrl = readState<string>(props.state?.mediaUrl)
    if (typeof mediaUrl === 'string' && mediaUrl) return mediaUrl
    return props.path || props.file || ''
  })

  const isPlaying = computed(() => {
    const statePlaying = readState<boolean>(props.state?.isPlaying)
    if (typeof statePlaying === 'boolean') {
      return statePlaying
    }
    return isLocalPlaying.value
  })

  const currentPosition = computed(() => {
    const now = readState<number>(props.state?.now)
    if (typeof now === 'number') {
      return now
    }
    return localPosition.value
  })

  const totalDuration = computed(() => {
    const runs = readState<number>(props.state?.runs)
    if (typeof runs === 'number') {
      return runs
    }
    const duration = readState<number>(props.state?.duration)
    if (typeof duration === 'number') {
      return duration
    }
    return localDuration.value
  })

  watch(audioSource, (newUrl) => {
    if (audioElement.value && newUrl) {
      audioElement.value.src = newUrl
      audioElement.value.load()
    }
  })

  onMounted(() => {
    if (audioElement.value && audioSource.value) {
      audioElement.value.src = audioSource.value
    }
  })

  onUnmounted(() => {
    if (audioElement.value) {
      audioElement.value.pause()
    }
  })

  function onPlay() {
    if (props.state?.play) {
      props.state.play()
    } else if (audioElement.value) {
      void audioElement.value.play()
    }
    isLocalPlaying.value = true
    emit('play')
  }

  function onPause() {
    if (props.state?.pause) {
      props.state.pause()
    } else if (audioElement.value) {
      audioElement.value.pause()
    }
    isLocalPlaying.value = false
    emit('pause')
  }

  function onTogglePlay() {
    if (isPlaying.value) {
      onPause()
    } else {
      onPlay()
    }
  }

  function onSeekMs(ms: number) {
    const safeMs = Number.isFinite(ms) ? ms : 0
    const total = Number.isFinite(totalDuration.value) ? totalDuration.value : 0
    const clamped = Math.max(0, Math.min(safeMs, total))
    if (props.state?.go) {
      props.state.go(clamped)
    } else if (props.state?.seek) {
      props.state.seek(clamped)
    }
    if (audioElement.value) {
      audioElement.value.currentTime = clamped / 1000
    }
    localPosition.value = clamped
    emit('seek', clamped)
  }

  function onToggleMute() {
    isMuted.value = !isMuted.value
    if (audioElement.value) {
      audioElement.value.muted = isMuted.value
    }
  }

  function onAudioTimeUpdate() {
    if (audioElement.value) {
      localPosition.value = Math.round(audioElement.value.currentTime * 1000)
    }
  }

  function onAudioLoadedMetadata() {
    if (audioElement.value && !totalDuration.value) {
      localDuration.value = Math.round(audioElement.value.duration * 1000)
    }
  }

  function onAudioEnded() {
    isLocalPlaying.value = false
    localPosition.value = totalDuration.value
  }

  watch(
    audioElement,
    (el, oldEl) => {
      if (oldEl) {
        oldEl.removeEventListener('timeupdate', onAudioTimeUpdate)
        oldEl.removeEventListener('loadedmetadata', onAudioLoadedMetadata)
        oldEl.removeEventListener('ended', onAudioEnded)
      }
      if (el) {
        el.addEventListener('timeupdate', onAudioTimeUpdate)
        el.addEventListener('loadedmetadata', onAudioLoadedMetadata)
        el.addEventListener('ended', onAudioEnded)
      }
    },
    { immediate: true },
  )

  return {
    isPlaying,
    currentPosition,
    totalDuration,
    playbackRate,
    isMuted,
    onTogglePlay,
    onSeekMs,
    onToggleMute,
  }
}
