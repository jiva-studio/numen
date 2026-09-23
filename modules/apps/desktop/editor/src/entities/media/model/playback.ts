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

export function usePlayback(through: Player, plays: MediaTypeProbe, isOpen: () => boolean) {
  /** Where the recording's own bytes are played from, once it is asked. */
  const url = ref('')
  /** How long the recording runs, as the application last said. */
  const duration = ref(0)
  /** What the recording is played as, as the application answers it. */
  const type = ref('')
  /** The web address a url points at, and nothing on every other source. */
  const points = ref('')

  /**
   * The player drawn in the tab, which a url has and a recording does not: what
   * a url points at plays where it is drawn, not through the one the window
   * plays sound with.
   */
  const frame = shallowRef<TabPlayer | null>(null)
  /** Where that player stands, and nothing before it has said. */
  const framed = ref(-1)

  /**
   * A moment gone to before the recording knew its own address, played from
   * once it does. A hit in the words opens a tab and asks for a moment in the
   * same breath.
   */
  let wanted = -1

  /** Whether the recording the player holds is this one. */
  const held = computed(() => url.value !== '' && through.url.value === url.value)

  /**
   * How long the recording runs. The application says, and the recording
   * itself says where it is loaded and knows better.
   */
  const runs = computed(() => Math.max(duration.value, held.value ? through.duration.value : 0))

  /**
   * Where the player stands in this recording, in milliseconds.
   *
   * The window plays one recording at a time, so one the player is not holding
   * stands at its beginning until somebody plays it.
   */
  const now = computed(() => {
    if (frame.value) return Math.max(0, framed.value)
    return held.value ? through.at.value : 0
  })

  /** Whether the player has said where it stands in this recording. */
  const hasTime = computed(() => !(frame.value && framed.value < 0))

  /** Whether this recording is the one playing. */
  const isPlaying = computed(() => held.value && through.isPlaying.value)

  /** What the player could not do, while this is the recording it holds. */
  const broken = computed(() => (held.value ? through.error.value : ''))

  /** Whether this window can play a recording of this kind at all. */
  const playable = computed(() => url.value !== '' && plays(type.value))

  /**
   * Whether what is at the address is a page, which is read in a frame rather
   * than played. A url with no copy on this disk is one.
   */
  const framing = computed(() => url.value !== '' && type.value === PAGE)

  /** The tab hands over the player it drew, and says where it stands. */
  const setFramePlayer = (player: TabPlayer | null) => {
    frame.value = player
    framed.value = -1
  }
  const setFrameTime = (ms: number) => {
    framed.value = ms
  }

  /** The moment the person went to. Before the beginning is the beginning. */
  const go = (ms: number) => {
    if (!isOpen()) return
    const at = Math.max(0, Math.round(ms))
    if (frame.value) return frame.value.seek(at)
    if (!url.value) return void (wanted = at)
    through.seek(url.value, at)
  }

  /** Play this recording, taking the sound from whatever else held it. */
  const play = () => {
    if (!isOpen() || !url.value) return
    through.play(url.value)
  }

  /** Stop it, while it is this recording that is playing. */
  const pause = () => {
    if (isPlaying.value) through.pause()
  }

  /** What the recording is, as the application answers it. */
  const setSource = (summary: RecordingSummary) => {
    duration.value = summary.duration
    url.value = summary.mediaUrl
    type.value = summary.mediaType
    points.value = summary.url
    // A moment asked for before the recording knew where its bytes are.
    if (wanted >= 0 && url.value) {
      const at = wanted
      wanted = -1
      through.seek(url.value, at)
    }
    // The player holding nothing takes this recording, so the controls read
    // how long it runs before anybody presses play. One already in the
    // player is left where it is, and a url plays where it is drawn.
    if (!points.value && url.value && through.url.value === '') {
      through.load(url.value)
    }
  }

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
    setFramePlayer,
    setFrameTime,
    setSource,
    go,
    play,
    pause,
  }
}
