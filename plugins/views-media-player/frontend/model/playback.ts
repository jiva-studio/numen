/**
 * Playback state of one recording in the shared audio player.
 */
import { computed, ref, watch } from 'vue'
import type { MediaTypeProbe, Player } from './player'
import type { RecordingSource } from '../types'

/**
 * One recording's playback through the shared player. A moment gone to before the recording knows
 * its address is played from once it does.
 */
export function usePlayback(audioPlayer: Player, plays: MediaTypeProbe, isOpen: () => boolean) {
  const url = ref('')
  const type = ref('')
  const isSourceRead = ref(false)
  /** The length read of this recording's own source, for while another recording holds the player. */
  const ownDuration = ref(0)
  let pendingSeek = -1

  const isLoaded = computed(() => url.value !== '' && audioPlayer.url.value === url.value)
  const runs = computed(() =>
    isLoaded.value && audioPlayer.duration.value > 0
      ? audioPlayer.duration.value
      : ownDuration.value,
  )
  // A length the player read stays this recording's once another recording takes the player.
  watch(
    () => (isLoaded.value ? audioPlayer.duration.value : 0),
    (ms) => {
      if (ms > 0) ownDuration.value = ms
    },
  )
  const now = computed(() => (isLoaded.value ? audioPlayer.position.value : 0))
  const isPlaying = computed(() => isLoaded.value && audioPlayer.isPlaying.value)
  const failure = computed(() => (isLoaded.value ? audioPlayer.error.value : ''))
  const isPlayable = computed(() => url.value !== '' && plays(type.value))
  /** True once the source is known and its address or media type cannot play in this window. */
  const isUnplayable = computed(() => isSourceRead.value && !isPlayable.value)

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
    if (audioPlayer.url.value === '' || audioPlayer.url.value === source) return
    void audioPlayer.getDuration(source).then((ms) => {
      if (url.value === source) ownDuration.value = ms
    })
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
    } else if (audioPlayer.url.value === '') {
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
    url,
    runs,
    now,
    isPlaying,
    failure,
    isUnplayable,
    setSource,
    controls: { go, play, pause },
  }
}
