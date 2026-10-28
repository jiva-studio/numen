/**
 * The one sound the window makes, through a single element that belongs to no
 * tab and is not in the page, so a tab moved between panes keeps playing without a
 * gap.
 *
 * One sound plays at a time: a recording asked to play takes it from whatever
 * held it.
 */
import { doc } from '../dom'
import { readonly, ref, type Ref } from 'vue'
import { WORDS } from '../words'

// The codes a MediaError carries, under the names the standard gives them. A
// window with no media element of its own defines none of them.
const MEDIA_ERR_ABORTED = 1
const MEDIA_ERR_NETWORK = 2
const MEDIA_ERR_DECODE = 3
const MEDIA_ERR_SRC_NOT_SUPPORTED = 4
/** The readyState at which the element has read the metadata. */
const HAVE_METADATA = 1

/** What each of them is called where a person reads it. */
const FAILED: Record<number, string> = {
  [MEDIA_ERR_ABORTED]: WORDS.stopped,
  [MEDIA_ERR_NETWORK]: WORDS.unreached,
  [MEDIA_ERR_DECODE]: WORDS.undecoded,
  [MEDIA_ERR_SRC_NOT_SUPPORTED]: WORDS.unwanted,
}

/** The events after which the element may know a length it did not know before, besides play, timeupdate and seeked. */
const DURATION_EVENTS = [
  'durationchange',
  'loadedmetadata',
  'loadeddata',
  'canplay',
  'canplaythrough',
  'progress',
]

/** How long reading the length of a recording that does not hold the player may take. */
const DURATION_READ_TIMEOUT_MS = 20_000

/** How often, and how many times, a length still unknown once the metadata is read is asked for again. */
const DURATION_POLL_MS = 250
const DURATION_POLLS = 40

/** What plays a recording, as whatever draws the controls reads it. */
export interface Player {
  /** What is loaded now, and nothing where the window is silent. */
  readonly url: Readonly<Ref<string>>
  /** Where it stands, in milliseconds. */
  readonly position: Readonly<Ref<number>>
  /** How long it runs, in milliseconds, and zero until the recording says. */
  readonly duration: Readonly<Ref<number>>
  readonly isPlaying: Readonly<Ref<boolean>>
  /** What it could not do, in words a person reads. */
  readonly error: Readonly<Ref<string>>

  /** Load a recording, leaving whatever was loaded before it. */
  load(url: string): void
  play(url: string): void
  pause(): void
  /** Play a millisecond of the recording at an address, loading it first. */
  seek(url: string, ms: number): void
  /**
   * The length of the recording at an address in milliseconds, read through an element of its
   * own, so the recording that holds the player keeps it. Zero where it cannot be read.
   */
  getDuration(url: string): Promise<number>
}

/** What makes the element a sound is played through. A test says otherwise. */
export type AudioFactory = () => HTMLAudioElement

const createAudioElement: AudioFactory = () => new Audio()

/**
 * Reports the length of the element's recording in milliseconds, zero while it is unknown, each
 * time it may have changed. Gives back the function that reads it, for the events a caller
 * already listens to.
 *
 * WebKitGTK reports a VBR mp3 with no Xing header as 0 s long at durationchange and
 * loadedmetadata, then sets the estimated length without another durationchange. The length is
 * read again at each of these, and polled while it is still unknown.
 */
function watchDuration(audio: HTMLAudioElement, onDuration: (ms: number) => void): () => void {
  let poll: ReturnType<typeof setInterval> | undefined
  const stopPoll = () => {
    clearInterval(poll)
    poll = undefined
  }
  const readDuration = () => {
    const runs = audio.duration
    const ms = Number.isFinite(runs) && runs > 0 ? Math.round(runs * 1000) : 0
    onDuration(ms)
    if (ms > 0 || audio.readyState < HAVE_METADATA) {
      stopPoll()
      return
    }
    if (poll !== undefined) return
    const src = audio.src
    let remainingPolls = DURATION_POLLS
    poll = setInterval(() => {
      remainingPolls--
      if (audio.src !== src || remainingPolls <= 0) stopPoll()
      else readDuration()
    }, DURATION_POLL_MS)
  }
  for (const name of DURATION_EVENTS) audio.addEventListener(name, readDuration)
  return readDuration
}

function bindAudioListeners(
  audio: HTMLAudioElement,
  state: {
    position: Ref<number>
    duration: Ref<number>
    isPlaying: Ref<boolean>
    error: Ref<string>
  },
) {
  audio.preload = 'metadata'
  const readDuration = watchDuration(audio, (ms) => (state.duration.value = ms))
  const readPosition = () => {
    state.position.value = Math.round(audio.currentTime * 1000)
    readDuration()
  }
  audio.addEventListener('timeupdate', readPosition)
  audio.addEventListener('seeked', readPosition)
  audio.addEventListener('play', () => {
    state.isPlaying.value = true
    readDuration()
  })
  audio.addEventListener('pause', () => void (state.isPlaying.value = false))
  audio.addEventListener('ended', () => void (state.isPlaying.value = false))
  audio.addEventListener('error', () => {
    state.error.value = FAILED[audio.error?.code ?? 0] ?? WORDS.unreadable
  })
}

/**
 * One sound, played through one element made when it is first wanted. The
 * element is never put in the page, so nothing that is drawn can take it away.
 */
export function createAudioPlayer(create: AudioFactory = createAudioElement): Player {
  const url = ref('')
  const position = ref(0)
  const duration = ref(0)
  const isPlaying = ref(false)
  const error = ref('')

  let element: HTMLAudioElement | null = null

  const getElement = (): HTMLAudioElement | null => {
    if (element) return element
    let audio: HTMLAudioElement
    try {
      audio = create()
    } catch {
      // A window that cannot make the element plays nothing, and says so.
      error.value = WORDS.unreadable
      return null
    }
    element = audio
    bindAudioListeners(audio, { position, duration, isPlaying, error })
    return audio
  }

  /** Put a recording in the player, and give the element it is in, or null where it is not there. */
  const load = (next: string): HTMLAudioElement | null => {
    const audio = getElement()
    if (!audio || !next) return null
    if (url.value === next) return audio
    audio.src = next
    url.value = next
    position.value = 0
    duration.value = 0
    isPlaying.value = false
    error.value = ''
    return audio
  }

  /** The lengths read of recordings that do not hold the player, by address. */
  const lengths = new Map<string, Promise<number>>()

  const readLength = (next: string): Promise<number> =>
    new Promise((resolve) => {
      let probe: HTMLAudioElement
      try {
        probe = create()
      } catch {
        resolve(0)
        return
      }
      let isDone = false
      const finish = (ms: number) => {
        if (isDone) return
        isDone = true
        clearTimeout(timer)
        // The element lets go of the recording's connection once its source is gone.
        probe.removeAttribute?.('src')
        probe.load?.()
        resolve(ms)
      }
      const timer = setTimeout(() => finish(0), DURATION_READ_TIMEOUT_MS)
      probe.preload = 'metadata'
      watchDuration(probe, (ms) => {
        if (ms > 0) finish(ms)
      })
      probe.addEventListener('error', () => finish(0))
      probe.src = next
    })

  const getDuration = (next: string): Promise<number> => {
    if (!next) return Promise.resolve(0)
    const cached = lengths.get(next)
    if (cached) return cached
    const length = readLength(next)
    lengths.set(next, length)
    void length.then((ms) => {
      if (ms === 0) lengths.delete(next)
    })
    return length
  }

  return {
    url: readonly(url),
    position: readonly(position),
    duration: readonly(duration),
    isPlaying: readonly(isPlaying),
    error: readonly(error),

    load: (url) => void load(url),

    play: (url) => {
      const audio = load(url)
      if (!audio) return
      void audio.play().catch((thrown: unknown) => {
        isPlaying.value = false
        // A play the window itself cut short is not a failure. Anything else is
        // a press that did nothing, and no `error` event fires on it, so this
        // is the only place it can be said.
        if ((thrown as { name?: string } | null)?.name === 'AbortError') return
        if (error.value === '') error.value = WORDS.unreadable
      })
    },

    pause: () => element?.pause(),

    seek: (url, ms) => {
      const audio = load(url)
      if (!audio) return
      const to = Math.max(0, Math.round(ms))
      position.value = to
      audio.currentTime = to / 1000
    },

    getDuration,
  }
}

/** The one player this window has. Every recording is played through it. */
export const player: Player = createAudioPlayer()

/** What is asked whether a kind of sound can be played. */
export type MediaTypeProbe = (type: string) => boolean

/** What the window itself says about a kind of sound. */
const probeWindow: MediaTypeProbe = (type) => {
  try {
    const mediaType = type === 'audio/vnd.wave' ? 'audio/wav' : type
    return doc.createElement('audio').canPlayType(mediaType) !== ''
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
export function createMediaTypeProbe(answers: MediaTypeProbe = probeWindow): MediaTypeProbe {
  const answersByType = new Map<string, boolean>()
  return (type) => {
    if (!type) return false
    const answer = answersByType.get(type)
    if (answer !== undefined) return answer
    const can = answers(type)
    answersByType.set(type, can)
    return can
  }
}
