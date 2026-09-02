/**
 * The one sound the window makes.
 *
 * A recording is played through a single element that belongs to no tab and
 * stands in no page. A tab moved between panes is drawn again; the sound is
 * not, and goes on through the move without a gap.
 *
 * One sound plays at a time. A recording asked to play takes it from whatever
 * held it, which is what a person means by pressing play on a second one.
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
export type Makes = () => HTMLAudioElement

const made: Makes = () => new Audio()

/**
 * One sound, played through one element made when it is first wanted. The
 * element is never put in the page, so nothing that is drawn can take it away.
 */
export function audio(makes: Makes = made): Player {
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
    element.addEventListener('play', () => void (playing.value =true))
    element.addEventListener('pause', () => void (playing.value =false))
    element.addEventListener('ended', () => void (playing.value =false))
    element.addEventListener('error', () => {
      failed.value = FAILED[element?.error?.code ?? 0] ?? WORDS.unreadable
    })
    return element
  }

  /** Put a recording in the sound, and say whether it is there. */
  const load = (wanted: string): boolean => {
    const sound = held()
    if (!sound || !wanted) return false
    if (address.value === wanted) return true
    sound.src = wanted
    address.value = wanted
    at.value = 0
    length.value = 0
    playing.value =false
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
        playing.value =false
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
