/**
 * The one player the window has.
 *
 * The element is made when it is first wanted and never put in the page, so
 * what it does is asked of it here through a stand-in element.
 */
import { describe, expect, it } from 'vitest'
import { audio, type AudioFactory } from './player'
import { WORDS } from './words'

const TALK = 'http://127.0.0.1:1/files/w/v/talk.mp3'
const OTHER = 'http://127.0.0.1:1/files/w/v/other.mp3'

/**
 * An element as far as the player uses one. It records what was asked of it
 * and fires the events a real one fires, when a test says to.
 */
function stands() {
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
    /** What a call to play answers with, and what it refuses with. */
    refuses: false,
    refusal: new Error('the window would not') as Error,

    addEventListener(name: string, run: () => void) {
      const held = listeners.get(name) ?? []
      held.push(run)
      listeners.set(name, held)
    },

    play() {
      element.started++
      if (element.refuses) return Promise.reject(element.refusal)
      element.paused = false
      fires('play')
      return Promise.resolve()
    },

    pause() {
      element.stopped++
      element.paused = true
      fires('pause')
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
  const fires = (name: string) => {
    for (const run of listeners.get(name) ?? []) run()
  }

  /** How many are listening for an event. */
  const listening = (name: string) => (listeners.get(name) ?? []).length

  /** The recording says how long it runs. */
  const runs = (seconds: number) => {
    element.duration = seconds
    fires('durationchange')
  }

  /** The recording plays on, and says where it stands. */
  const moves = (seconds: number) => {
    element.currentTime = seconds
    fires('timeupdate')
  }

  /** The element could not do it, and says which of the four it was. */
  const breaks = (code: number) => {
    element.error = { code }
    fires('error')
  }

  return { element, fires, listening, runs, moves, breaks }
}

/** A player standing on one element, and the element it stands on. */
const player = () => {
  const stood = stands()
  const makes: AudioFactory = () => stood.element as unknown as HTMLAudioElement
  return { plays: audio(makes), ...stood }
}

describe('a recording loaded', () => {
  it('is where the player stands, and nothing has been asked of it', () => {
    const { plays, element } = player()

    plays.load(TALK)

    expect(plays.address.value).toBe(TALK)
    expect(element.loaded).toStrictEqual([TALK])
    expect(element.started).toBe(0)
    expect(plays.playing.value).toBe(false)
  })

  it('is loaded once, however often it is asked for', () => {
    const { plays, element, moves } = player()
    plays.load(TALK)
    moves(4)

    plays.load(TALK)

    expect(element.loaded).toStrictEqual([TALK])
    expect(plays.at.value).toBe(4_000)
  })

  it('is nothing at all where the application named no address', () => {
    const { plays, element } = player()

    plays.load('')

    expect(plays.address.value).toBe('')
    expect(element.loaded).toStrictEqual([])
  })

  it('leaves the moment it stood at behind when another takes the player', () => {
    const { plays, moves } = player()
    plays.load(TALK)
    moves(6)

    plays.load(OTHER)

    expect(plays.address.value).toBe(OTHER)
    expect(plays.at.value).toBe(0)
    expect(plays.length.value).toBe(0)
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
    expect(plays.address.value).toBe(OTHER)
    expect(plays.playing.value).toBe(true)
  })

  it('is not playing where the window refused it, and says so', async () => {
    const { plays, element } = player()
    element.refuses = true

    plays.play(TALK)
    await Promise.resolve()
    await Promise.resolve()

    expect(plays.playing.value).toBe(false)
    expect(plays.failed.value).toBe(WORDS.unreadable)
  })

  it('says nothing of a play the window itself cut short', async () => {
    const { plays, element } = player()
    element.refuses = true
    element.refusal = Object.assign(new Error('another recording took it'), {
      name: 'AbortError',
    })

    plays.play(TALK)
    await Promise.resolve()
    await Promise.resolve()

    expect(plays.failed.value).toBe('')
  })

  it('stops where the person stops it, and stands where it stopped', () => {
    const { plays, moves } = player()
    plays.play(TALK)
    moves(3)

    plays.pause()

    expect(plays.playing.value).toBe(false)
    expect(plays.at.value).toBe(3_000)
  })

  it('stops of itself when it ends', () => {
    const { plays, fires } = player()
    plays.play(TALK)

    fires('ended')

    expect(plays.playing.value).toBe(false)
  })
})

describe('how long a recording runs', () => {
  it('is what the recording says, in milliseconds', () => {
    const { plays, runs } = player()
    plays.load(TALK)

    runs(85.25)

    expect(plays.length.value).toBe(85_250)
  })

  it('is nothing where the recording never ends', () => {
    const { plays, runs } = player()
    plays.load(TALK)

    runs(Number.POSITIVE_INFINITY)

    expect(plays.length.value).toBe(0)
  })

  it('is nothing where the recording has not said', () => {
    const { plays, runs } = player()
    plays.load(TALK)

    runs(Number.NaN)

    expect(plays.length.value).toBe(0)
  })
})

describe('a recording the player could not read', () => {
  it('says which of the four it was, in words a person reads', () => {
    const { plays, breaks } = player()
    plays.load(TALK)

    breaks(2)

    expect(plays.failed.value).toBe(WORDS.unreached)
  })

  it('says the recording could not be played where it named no code', () => {
    const { plays, breaks } = player()
    plays.load(TALK)

    breaks(9)

    expect(plays.failed.value).toBe(WORDS.unreadable)
  })

  it('says nothing again once another recording takes the player', () => {
    const { plays, breaks } = player()
    plays.load(TALK)
    breaks(4)

    plays.load(OTHER)

    expect(plays.failed.value).toBe('')
  })
})

describe('the element a window plays through', () => {
  it('is made once, and listened to once, however many recordings are opened', () => {
    let made = 0
    const stood = stands()
    const plays = audio(() => {
      made++
      return stood.element as unknown as HTMLAudioElement
    })

    for (let at = 0; at < 100; at++) plays.load(`http://127.0.0.1:1/files/w/v/${at}.mp3`)

    expect(made).toBe(1)
    expect(stood.listening('timeupdate')).toBe(1)
    expect(stood.listening('error')).toBe(1)
  })

  it('plays nothing and says so where the window has no element to give', () => {
    const plays = audio(() => {
      throw new Error('this window plays no sound')
    })

    plays.load(TALK)
    plays.play(TALK)
    plays.seek(TALK, 1_000)
    plays.pause()

    expect(plays.address.value).toBe('')
    expect(plays.playing.value).toBe(false)
    expect(plays.failed.value).toBe(WORDS.unreadable)
  })
})
