/**
 * Playback state of one recording in the shared audio player.
 */
import { computed, ref } from 'vue'
import type { MediaTypeProbe, Player } from './player'
import type { RecordingSource } from '../types'

/**
 * One recording's playback through the shared player. A moment gone to before the recording knows
 * its address is played from once it does.
 */
export function usePlayback(audioPlayer: Player, plays: MediaTypeProbe, isOpen: () => boolean) {
  const url = ref('')
  const type = ref('')
  let pendingSeek = -1

  const isLoaded = computed(() => url.value !== '' && audioPlayer.url.value === url.value)
  const runs = computed(() => (isLoaded.value ? audioPlayer.duration.value : 0))
  const now = computed(() => (isLoaded.value ? audioPlayer.position.value : 0))
  const isPlaying = computed(() => isLoaded.value && audioPlayer.isPlaying.value)
  const failure = computed(() => (isLoaded.value ? audioPlayer.error.value : ''))
  const isPlayable = computed(() => url.value !== '' && plays(type.value))

  const go = (ms: number) => {
    if (!isOpen()) return
    const position = Math.max(0, Math.round(ms))
    if (!url.value) {
      pendingSeek = position
      return
    }
    audioPlayer.seek(url.value, position)
  }

  const setSource = (source: RecordingSource) => {
    url.value = source.mediaUrl
    type.value = source.mediaType
    if (!url.value) return
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
    isPlayable,
    setSource,
    controls: { go, play, pause },
  }
}
