/**
 * The one player the window has.
 *
 * The element is made when it is first wanted and never put in the page, so
 * what it does is asked of it here through a stand-in element.
 */
import { describe, expect, it } from 'vitest'
import { createAudioPlayer, createMediaTypeProbe, type AudioFactory } from './player'
import { WORDS } from '../words'

const TALK = 'http://127.0.0.1:1/files/w/v/talk.mp3'
const OTHER = 'http://127.0.0.1:1/files/w/v/other.mp3'

/**
 * An element as far as the player uses one. It records what was asked of it
 * and fires the events a real one fires, when a test says to.
 */
function createElement() {
  const listeners = new Map<string, (() => void)[]>()

  const element = {
    src: '',
    preload: '',
    currentTime: 0,
    duration: Number.NaN,
    paused: true,
    error: null as { code: number } | null,
    /** Every address it was given, in the order it was given them. */
    loaded: [] as string[],
    /** How many times it was asked to play, and to stop. */
    started: 0,
    stopped: 0,
    /** What a call to play answers with, and what it fails with. */
    fails: false,
    playError: new Error('the window would not') as Error,

    addEventListener(name: string, run: () => void) {
      const held = listeners.get(name) ?? []
      held.push(run)
      listeners.set(name, held)
    },

    play() {
      element.started++
      if (element.fails) return Promise.reject(element.playError)
      element.paused = false
      emit('play')
      return Promise.resolve()
    },

    pause() {
      element.stopped++
      element.paused = true
      emit('pause')
    },
  }

  // The address is written down as it is set, the way a real element begins
  // loading as it is set.
  const source = { value: '' }
  Object.defineProperty(element, 'src', {
    get: () => source.value,
    set: (wanted: string) => {
      source.value = wanted
      element.loaded.push(wanted)
    },
  })

  /** One event, as the element fires it. */
  const emit = (name: string) => {
    for (const run of listeners.get(name) ?? []) run()
  }

  /** How many are listening for an event. */
  const countListeners = (name: string) => (listeners.get(name) ?? []).length

  /** The recording says how long it runs. */
  const setDuration = (seconds: number) => {
    element.duration = seconds
    emit('durationchange')
  }

  /** The recording plays on, and says where it stands. */
  const setCurrentTime = (seconds: number) => {
    element.currentTime = seconds
    emit('timeupdate')
  }

  /** The element could not do it, and says which of the four it was. */
  const setError = (code: number) => {
    element.error = { code }
    emit('error')
  }

  return { element, emit, countListeners, setDuration, setCurrentTime, setError }
}

/** A player standing on one element, and the element it stands on. */
const player = () => {
  const stood = createElement()
  const getElement: AudioFactory = () => stood.element as unknown as HTMLAudioElement
  return { plays: createAudioPlayer(getElement), ...stood }
}

describe('a recording loaded', () => {
  it('is where the player stands, and nothing has been asked of it', () => {
    const { plays, element } = player()

    plays.load(TALK)

    expect(plays.url.value).toBe(TALK)
    expect(element.loaded).toStrictEqual([TALK])
    expect(element.started).toBe(0)
    expect(plays.playing.value).toBe(false)
  })

  it('is loaded once, however often it is asked for', () => {
    const { plays, element, setCurrentTime } = player()
    plays.load(TALK)
    setCurrentTime(4)

    plays.load(TALK)

    expect(element.loaded).toStrictEqual([TALK])
    expect(plays.at.value).toBe(4_000)
  })

  it('is nothing at all where the application named no address', () => {
    const { plays, element } = player()

    plays.load('')

    expect(plays.url.value).toBe('')
    expect(element.loaded).toStrictEqual([])
  })

  it('leaves the moment it stood at behind when another takes the player', () => {
    const { plays, setCurrentTime } = player()
    plays.load(TALK)
    setCurrentTime(6)

    plays.load(OTHER)

    expect(plays.url.value).toBe(OTHER)
    expect(plays.at.value).toBe(0)
    expect(plays.duration.value).toBe(0)
  })
})

describe('a moment gone to', () => {
  it('loads the recording first where nothing is loaded', () => {
    const { plays, element } = player()

    plays.seek(TALK, 2_500)

    expect(element.loaded).toStrictEqual([TALK])
    expect(element.currentTime).toBe(2.5)
    expect(plays.at.value).toBe(2_500)
  })

  it('is the beginning when it is before that', () => {
    const { plays, element } = player()

    plays.seek(TALK, -400)

    expect(element.currentTime).toBe(0)
    expect(plays.at.value).toBe(0)
  })

  it('is nowhere at all where the application named no address', () => {
    const { plays, element } = player()

    plays.seek('', 2_500)

    expect(element.loaded).toStrictEqual([])
    expect(plays.at.value).toBe(0)
  })
})

describe('a recording played', () => {
  it('takes the player from whatever held it', async () => {
    const { plays, element } = player()
    plays.play(TALK)
    await Promise.resolve()

    plays.play(OTHER)
    await Promise.resolve()

    expect(element.loaded).toStrictEqual([TALK, OTHER])
    expect(plays.url.value).toBe(OTHER)
    expect(plays.playing.value).toBe(true)
  })

  it('is not playing where the window refused it, and says so', async () => {
    const { plays, element } = player()
    element.fails = true

    plays.play(TALK)
    await Promise.resolve()
    await Promise.resolve()

    expect(plays.playing.value).toBe(false)
    expect(plays.error.value).toBe(WORDS.unreadable)
  })

  it('says nothing of a play the window itself cut short', async () => {
    const { plays, element } = player()
    element.fails = true
    element.playError = Object.assign(new Error('another recording took it'), {
      name: 'AbortError',
    })

    plays.play(TALK)
    await Promise.resolve()
    await Promise.resolve()

    expect(plays.error.value).toBe('')
  })

  it('stops where the person stops it, and stands where it stopped', () => {
    const { plays, setCurrentTime } = player()
    plays.play(TALK)
    setCurrentTime(3)

    plays.pause()

    expect(plays.playing.value).toBe(false)
    expect(plays.at.value).toBe(3_000)
  })

  it('stops of itself when it ends', () => {
    const { plays, emit } = player()
    plays.play(TALK)

    emit('ended')

    expect(plays.playing.value).toBe(false)
  })
})

describe('how long a recording runs', () => {
  it('is what the recording says, in milliseconds', () => {
    const { plays, setDuration } = player()
    plays.load(TALK)

    setDuration(85.25)

    expect(plays.duration.value).toBe(85_250)
  })

  it('is nothing where the recording never ends', () => {
    const { plays, setDuration } = player()
    plays.load(TALK)

    setDuration(Number.POSITIVE_INFINITY)

    expect(plays.duration.value).toBe(0)
  })

  it('is nothing where the recording has not said', () => {
    const { plays, setDuration } = player()
    plays.load(TALK)

    setDuration(Number.NaN)

    expect(plays.duration.value).toBe(0)
  })
})

describe('a recording the player could not read', () => {
  it('says which of the four it was, in words a person reads', () => {
    const { plays, setError } = player()
    plays.load(TALK)

    setError(2)

    expect(plays.error.value).toBe(WORDS.unreached)
  })

  it('says the recording could not be played where it named no code', () => {
    const { plays, setError } = player()
    plays.load(TALK)

    setError(9)

    expect(plays.error.value).toBe(WORDS.unreadable)
  })

  it('says nothing again once another recording takes the player', () => {
    const { plays, setError } = player()
    plays.load(TALK)
    setError(4)

    plays.load(OTHER)

    expect(plays.error.value).toBe('')
  })
})

describe('the element a window plays through', () => {
  it('is made once, and listened to once, however many recordings are opened', () => {
    let made = 0
    const stood = createElement()
    const plays = createAudioPlayer(() => {
      made++
      return stood.element as unknown as HTMLAudioElement
    })

    for (let at = 0; at < 100; at++) plays.load(`http://127.0.0.1:1/files/w/v/${at}.mp3`)

    expect(made).toBe(1)
    expect(stood.countListeners('timeupdate')).toBe(1)
    expect(stood.countListeners('error')).toBe(1)
  })

  it('plays nothing and says so where the window has no element to give', () => {
    const plays = createAudioPlayer(() => {
      throw new Error('this window plays no sound')
    })

    plays.load(TALK)
    plays.play(TALK)
    plays.seek(TALK, 1_000)
    plays.pause()

    expect(plays.url.value).toBe('')
    expect(plays.playing.value).toBe(false)
    expect(plays.error.value).toBe(WORDS.unreadable)
  })
})

describe('what this window can play', () => {
  it('asks the window once, however many recordings are open', () => {
    let asks = 0
    const plays = createMediaTypeProbe((type) => {
      asks++
      return type === 'audio/mpeg'
    })

    expect(plays('audio/mpeg')).toBe(true)
    expect(plays('audio/mpeg')).toBe(true)
    expect(plays('audio/mpeg')).toBe(true)
    expect(asks).toBe(1)

    expect(plays('audio/wav')).toBe(false)
    expect(asks).toBe(2)
  })

  it('plays nothing where the application named no type', () => {
    expect(createMediaTypeProbe(() => true)('')).toBe(false)
  })

  it('plays nothing where the window answers for nothing', () => {
    expect(createMediaTypeProbe(() => false)('audio/mpeg')).toBe(false)
  })
})
