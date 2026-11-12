/**
 * Playback state of one recording in the shared audio player.
 */
import { computed, ref, watch } from 'vue'
import type { MediaTypeProbe, Player } from './player'
import type { RecordingSource } from '../types'

/**
 * One recording's playback through the shared player. A seek issued before the recording's address
 * is known is applied once the address is available.
 */
export function usePlayback(audioPlayer: Player, plays: MediaTypeProbe, isOpen: () => boolean) {
  const url = ref('')
  const type = ref('')
  const isSourceRead = ref(false)
  /** The length read of this recording's own source, for while another recording holds the player. */
  const ownDuration = ref(0)
  let pendingSeek = -1

  const isLoaded = computed(() => url.value !== '' && audioPlayer.state.url.value === url.value)
  const runs = computed(() =>
    isLoaded.value && audioPlayer.state.duration.value > 0
      ? audioPlayer.state.duration.value
      : ownDuration.value,
  )
  // A length the player read stays this recording's once another recording takes the player.
  watch(
    () => (isLoaded.value ? audioPlayer.state.duration.value : 0),
    (ms) => {
      if (ms > 0) ownDuration.value = ms
    },
  )
  const now = computed(() => (isLoaded.value ? audioPlayer.state.position.value : 0))
  const isPlaying = computed(() => isLoaded.value && audioPlayer.state.isPlaying.value)
  const failure = computed(() => (isLoaded.value ? audioPlayer.state.error.value : ''))
  const isPlayable = computed(() => url.value !== '' && plays(type.value))
  /** True once the source is known and its address or media type cannot play in this window. */
  const isUnplayable = computed(() => isSourceRead.value && !isPlayable.value)
  /** Whether the source is a video, whose picture the tab draws while it holds the player. */
  const isVideo = computed(() => type.value.startsWith('video/'))

  const go = (ms: number) => {
    if (!isOpen()) return
    const position = Math.max(0, Math.round(ms))
    if (!url.value) {
      pendingSeek = position
      return
    }
    audioPlayer.seek(url.value, position)
  }

  const readOwnDuration = (source: string) => {
    if (audioPlayer.state.url.value === '' || audioPlayer.state.url.value === source) return
    audioPlayer.getDuration(source).then(
      (ms) => {
        if (url.value === source) ownDuration.value = ms
      },
      // A duration that cannot be read leaves the own duration unset.
      () => undefined,
    )
  }

  const setSource = (source: RecordingSource) => {
    isSourceRead.value = true
    url.value = source.mediaUrl
    type.value = source.mediaType
    if (!url.value) return
    readOwnDuration(url.value)
    if (pendingSeek >= 0) {
      audioPlayer.seek(url.value, pendingSeek)
      pendingSeek = -1
    } else if (audioPlayer.state.url.value === '') {
      audioPlayer.load(url.value)
    }
  }

  const play = () => {
    if (!isOpen() || !url.value) return
    audioPlayer.play(url.value)
  }

  const pause = () => {
    if (isPlaying.value) audioPlayer.pause()
  }

  return {
    player: audioPlayer,
    source: { url, isLoaded, isVideo, isUnplayable, setSource },
    progress: { runs, now, isPlaying, failure },
    controls: { go, play, pause },
  }
}
