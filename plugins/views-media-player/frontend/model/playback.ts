/**
 * Where one recording stands in the single player the window makes sound with.
 */
import { computed, ref } from 'vue'
import type { MediaTypeProbe, Player } from './player'
import type { RecordingSource } from '../types'

/**
 * One recording's playback through the shared player. A moment gone to before the recording knows
 * its address is played from once it does.
 */
export function usePlayback(through: Player, plays: MediaTypeProbe, isOpen: () => boolean) {
  const url = ref('')
  const type = ref('')
  let pendingSeek = -1

  const isLoaded = computed(() => url.value !== '' && through.url.value === url.value)
  const runs = computed(() => (isLoaded.value ? through.duration.value : 0))
  const now = computed(() => (isLoaded.value ? through.at.value : 0))
  const isPlaying = computed(() => isLoaded.value && through.isPlaying.value)
  const failure = computed(() => (isLoaded.value ? through.error.value : ''))
  const isPlayable = computed(() => url.value !== '' && plays(type.value))

  const go = (ms: number) => {
    if (!isOpen()) return
    const at = Math.max(0, Math.round(ms))
    if (!url.value) {
      pendingSeek = at
      return
    }
    through.seek(url.value, at)
  }

  const setSource = (source: RecordingSource) => {
    url.value = source.mediaUrl
    type.value = source.mediaType
    if (!url.value) return
    if (pendingSeek >= 0) {
      through.seek(url.value, pendingSeek)
      pendingSeek = -1
    } else if (through.url.value === '') {
      through.load(url.value)
    }
  }

  const play = () => {
    if (!isOpen() || !url.value) return
    through.play(url.value)
  }

  const pause = () => {
    if (isPlaying.value) through.pause()
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
