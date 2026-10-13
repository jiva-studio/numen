/**
 * Where one recording stands: in the single player the window makes sound
 * with, or in the frame a url draws for itself.
 */
import { computed, ref, shallowRef } from 'vue'
import { type MediaTypeProbe, type Player } from './player'
import type { RecordingSummary } from '../types'

/** The player a tab drew for itself, which a moment chosen in the words seeks. */
export interface TabPlayer {
  seek(ms: number): void
}

/** What a url with no copy on this disk is played as: a page, read in a frame. */
const PAGE = 'text/html'

function createPlaybackActions(
  through: Player,
  refs: {
    url: Ref<string>
    type: Ref<string>
    duration: Ref<number>
    points: Ref<string>
    frame: ShallowRef<TabPlayer | null>
    framed: Ref<number>
    isPlaying: () => boolean
  },
  isOpen: () => boolean,
) {
  let wanted = -1

  const setFramePlayer = (player: TabPlayer | null) => {
    refs.frame.value = player
    if (player && wanted >= 0) {
      const at = wanted
      wanted = -1
      refs.framed.value = at
      player.seek(at)
    } else {
      refs.framed.value = -1
    }
  }

  const go = (ms: number) => {
    if (!isOpen()) return
    const at = Math.max(0, Math.round(ms))
    if (refs.frame.value) {
      refs.framed.value = at
      return refs.frame.value.seek(at)
    }
    if (!refs.url.value || refs.points.value !== '') return void (wanted = at)
    through.seek(refs.url.value, at)
  }

  const setSource = (summary: RecordingSummary) => {
    refs.duration.value = summary.duration
    refs.url.value = summary.mediaUrl
    refs.type.value = summary.mediaType
    refs.points.value = summary.url
    if (wanted >= 0) {
      const at = wanted
      if (refs.frame.value) {
        wanted = -1
        refs.framed.value = at
        refs.frame.value.seek(at)
      } else if (refs.url.value && !refs.points.value) {
        wanted = -1
        through.seek(refs.url.value, at)
      }
    }
    if (!refs.points.value && refs.url.value && through.url.value === '') {
      through.load(refs.url.value)
    }
  }

  return {
    setFramePlayer,
    setFrameTime: (ms: number) => {
      refs.framed.value = ms
    },
    go,
    play: () => {
      if (!isOpen() || !refs.url.value) return
      through.play(refs.url.value)
    },
    pause: () => {
      if (refs.isPlaying()) through.pause()
    },
    setSource,
  }
}

export function usePlayback(through: Player, plays: MediaTypeProbe, isOpen: () => boolean) {
  const url = ref('')
  const duration = ref(0)
  const type = ref('')
  const points = ref('')
  const frame = shallowRef<TabPlayer | null>(null)
  const framed = ref(-1)

  const held = computed(() => url.value !== '' && through.url.value === url.value)
  const runs = computed(() => Math.max(duration.value, held.value ? through.duration.value : 0))
  const now = computed(() => {
    if (frame.value) return Math.max(0, framed.value)
    return held.value ? through.at.value : 0
  })
  const hasTime = computed(() => !(frame.value && framed.value < 0))
  const isPlaying = computed(() => held.value && through.isPlaying.value)
  const broken = computed(() => (held.value ? through.error.value : ''))
  const playable = computed(() => url.value !== '' && plays(type.value))
  const framing = computed(() => url.value !== '' && type.value === PAGE)

  const actions = createPlaybackActions(
    through,
    {
      url,
      type,
      duration,
      points,
      frame,
      framed,
      isPlaying: () => isPlaying.value,
    },
    isOpen,
  )

  return {
    url,
    duration,
    type,
    points,
    runs,
    now,
    hasTime,
    isPlaying,
    broken,
    playable,
    framing,
    ...actions,
  }
}
