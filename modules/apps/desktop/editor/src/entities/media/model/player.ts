/**
 * The one sound the window makes, through a single element that belongs to no
 * tab and stands in no page, so a tab moved between panes plays on without a
 * gap.
 *
 * One sound plays at a time: a recording asked to play takes it from whatever
 * held it.
 */
import { readonly, ref, type Ref } from 'vue'
import { WORDS } from '../words'

// The codes a MediaError carries, under the names the standard gives them. A
// window with no media element of its own defines none of them.
const MEDIA_ERR_ABORTED = 1
const MEDIA_ERR_NETWORK = 2
const MEDIA_ERR_DECODE = 3
const MEDIA_ERR_SRC_NOT_SUPPORTED = 4

/** What each of them is called where a person reads it. */
const FAILED: Record<number, string> = {
  [MEDIA_ERR_ABORTED]: WORDS.stopped,
  [MEDIA_ERR_NETWORK]: WORDS.unreached,
  [MEDIA_ERR_DECODE]: WORDS.undecoded,
  [MEDIA_ERR_SRC_NOT_SUPPORTED]: WORDS.unwanted,
}

/** What plays a recording, as whatever draws the controls reads it. */
export interface Player {
  /** What is loaded now, and nothing where the window is silent. */
  readonly url: Readonly<Ref<string>>
  /** Where it stands, in milliseconds. */
  readonly at: Readonly<Ref<number>>
  /** How long it runs, in milliseconds, and zero until the recording says. */
  readonly duration: Readonly<Ref<number>>
  readonly playing: Readonly<Ref<boolean>>
  /** What it could not do, in words a person reads. */
  readonly error: Readonly<Ref<string>>

  /** Load a recording, leaving whatever was loaded before it. */
  load(url: string): void
  play(url: string): void
  pause(): void
  /** Play a millisecond of the recording at an address, loading it first. */
  seek(url: string, ms: number): void
}

/** What makes the element a sound is played through. A test says otherwise. */
export type AudioFactory = () => HTMLAudioElement

const createAudioElement: AudioFactory = () => new Audio()

/**
 * One sound, played through one element made when it is first wanted. The
 * element is never put in the page, so nothing that is drawn can take it away.
 */
export function createAudioPlayer(create: AudioFactory = createAudioElement): Player {
  const url = ref('')
  const at = ref(0)
  const duration = ref(0)
  const playing = ref(false)
  const error = ref('')

  let element: HTMLAudioElement | null = null

  const getElement = (): HTMLAudioElement | null => {
    if (element) return element
    try {
      element = create()
    } catch {
      // A window that cannot make the element plays nothing, and says so.
      error.value = WORDS.unreadable
      return null
    }
    element.preload = 'metadata'
    element.addEventListener('timeupdate', () => {
      at.value = Math.round((element?.currentTime ?? 0) * 1000)
    })
    element.addEventListener('seeked', () => {
      at.value = Math.round((element?.currentTime ?? 0) * 1000)
    })
    element.addEventListener('durationchange', () => {
      const runs = element?.duration ?? 0
      duration.value = Number.isFinite(runs) ? Math.round(runs * 1000) : 0
    })
    element.addEventListener('play', () => void (playing.value = true))
    element.addEventListener('pause', () => void (playing.value = false))
    element.addEventListener('ended', () => void (playing.value = false))
    element.addEventListener('error', () => {
      error.value = FAILED[element?.error?.code ?? 0] ?? WORDS.unreadable
    })
    return element
  }

  /** Put a recording in the player, and say whether it is there. */
  const load = (next: string): boolean => {
    const element = getElement()
    if (!element || !next) return false
    if (url.value === next) return true
    element.src = next
    url.value = next
    at.value = 0
    duration.value = 0
    playing.value = false
    error.value = ''
    return true
  }

  return {
    url: readonly(url),
    at: readonly(at),
    duration: readonly(duration),
    playing: readonly(playing),
    error: readonly(error),

    load: (url) => void load(url),

    play: (url) => {
      if (!load(url)) return
      void element?.play().catch((thrown: unknown) => {
        playing.value = false
        // A play the window itself cut short is not a failure. Anything else is
        // a press that did nothing, and no `error` event fires on it, so this
        // is the only place it can be said.
        if ((thrown as { name?: string } | null)?.name === 'AbortError') return
        if (error.value === '') error.value = WORDS.unreadable
      })
    },

    pause: () => element?.pause(),

    seek: (url, ms) => {
      if (!load(url)) return
      const to = Math.max(0, Math.round(ms))
      at.value = to
      if (element) element.currentTime = to / 1000
    },
  }
}

/** The one player this window has. Every recording is played through it. */
export const player: Player = createAudioPlayer()

/** What is asked whether a kind of sound can be played. */
export type MediaTypeProbe = (type: string) => boolean

/** What the window itself says about a kind of sound. */
const itself: MediaTypeProbe = (type) => {
  try {
    return document.createElement('audio').canPlayType(type) !== ''
  } catch {
    // A window that cannot be asked has not said yes, and what is offered is
    // what the window says it can play.
    return false
  }
}

/**
 * What this window can play. The answer is the window's own, and it is asked
 * once for each kind of sound however many recordings are open.
 */
export function createMediaTypeProbe(answers: MediaTypeProbe = itself): MediaTypeProbe {
  const asked = new Map<string, boolean>()
  return (type) => {
    if (!type) return false
    const held = asked.get(type)
    if (held !== undefined) return held
    const can = answers(type)
    asked.set(type, can)
    return can
  }
}
