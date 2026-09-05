/**
 * The one sound the window makes, through a single element that belongs to no
 * tab and stands in no page, so a tab moved between panes plays on without a
 * gap.
 *
 * One sound plays at a time: a recording asked to play takes it from whatever
 * held it.
 */
import { readonly, ref, type Ref } from 'vue'
import { WORDS } from './words'

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
  readonly address: Readonly<Ref<string>>
  /** Where it stands, in milliseconds. */
  readonly at: Readonly<Ref<number>>
  /** How long it runs, in milliseconds, and zero until the recording says. */
  readonly length: Readonly<Ref<number>>
  readonly playing: Readonly<Ref<boolean>>
  /** What it could not do, in words a person reads. */
  readonly failed: Readonly<Ref<string>>

  /** Load a recording, leaving whatever was loaded before it. */
  load(address: string): void
  play(address: string): void
  pause(): void
  /** Play a millisecond of the recording at an address, loading it first. */
  seek(address: string, ms: number): void
}

/** What makes the element a sound is played through. A test says otherwise. */
export type AudioFactory = () => HTMLAudioElement

const made: AudioFactory = () => new Audio()

/**
 * One sound, played through one element made when it is first wanted. The
 * element is never put in the page, so nothing that is drawn can take it away.
 */
export function audio(makes: AudioFactory = made): Player {
  const address = ref('')
  const at = ref(0)
  const length = ref(0)
  const playing = ref(false)
  const failed = ref('')

  let element: HTMLAudioElement | null = null

  const held = (): HTMLAudioElement | null => {
    if (element) return element
    try {
      element = makes()
    } catch {
      // A window that cannot make the element plays nothing at all, and every
      // press of play would otherwise do nothing and say nothing.
      failed.value = WORDS.unreadable
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
      length.value = Number.isFinite(runs) ? Math.round(runs * 1000) : 0
    })
    element.addEventListener('play', () => void (playing.value = true))
    element.addEventListener('pause', () => void (playing.value = false))
    element.addEventListener('ended', () => void (playing.value = false))
    element.addEventListener('error', () => {
      failed.value = FAILED[element?.error?.code ?? 0] ?? WORDS.unreadable
    })
    return element
  }

  /** Put a recording in the player, and say whether it is there. */
  const load = (wanted: string): boolean => {
    const element = held()
    if (!element || !wanted) return false
    if (address.value === wanted) return true
    element.src = wanted
    address.value = wanted
    at.value = 0
    length.value = 0
    playing.value = false
    failed.value = ''
    return true
  }

  return {
    address: readonly(address),
    at: readonly(at),
    length: readonly(length),
    playing: readonly(playing),
    failed: readonly(failed),

    load: (wanted) => void load(wanted),

    play: (wanted) => {
      if (!load(wanted)) return
      void element?.play().catch(() => {
        playing.value = false
      })
    },

    pause: () => element?.pause(),

    seek: (wanted, ms) => {
      if (!load(wanted)) return
      const to = Math.max(0, Math.round(ms))
      at.value = to
      if (element) element.currentTime = to / 1000
    },
  }
}

/** The one player this window has. Every recording is played through it. */
export const player: Player = audio()

/** What is asked whether a kind of sound can be played. */
export type MediaTypeProbe = (type: string) => boolean

/** Whether this window can play a recording of a media type. */
export type CanPlayType = (type: string) => boolean

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
export function playable(answers: MediaTypeProbe = itself): CanPlayType {
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
