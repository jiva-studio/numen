/**
 * The one player the window has.
 *
 * The element is created on first use and never attached to the page, so the
 * tests exercise it through a stand-in element.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createAudioPlayer, createMediaTypeProbe, type AudioFactory } from './player'
import { WORDS } from '../words'

const TALK = 'http://127.0.0.1:1/files/w/v/talk.mp3'
const OTHER = 'http://127.0.0.1:1/files/w/v/other.mp3'

/**
 * An element reduced to what the player uses. It records the calls made on it
 * and fires the events a real element fires, when a test requests it.
 */
function createElement() {
  const listeners = new Map<string, (() => void)[]>()

  const element = {
    src: '',
    preload: '',
    currentTime: 0,
    duration: Number.NaN,
    readyState: 0,
    paused: true,
    error: null as { code: number } | null,
    /** Every address it was given, in the order it was given them. */
    loaded: [] as string[],
    /** How many times play and stop were called. */
    started: 0,
    stopped: 0,
    /** What a call to play answers with, and what it fails with. */
    fails: false,
    playError: new Error('the window would not') as Error,

    addEventListener(name: string, run: () => void) {
      const handlers = listeners.get(name) ?? []
      handlers.push(run)
      listeners.set(name, handlers)
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

  /** The recording reports its duration. */
  const setDuration = (seconds: number) => {
    element.duration = seconds
    emit('durationchange')
  }

  /** Advances the element and emits timeupdate. */
  const setCurrentTime = (seconds: number) => {
    element.currentTime = seconds
    emit('timeupdate')
  }

  /** The element reports a failure with a media error code (one of four). */
  const setError = (code: number) => {
    element.error = { code }
    emit('error')
  }

  return { element, emit, countListeners, setDuration, setCurrentTime, setError }
}

/** A player over a fake element, and that element. */
const player = () => {
  const element = createElement()
  const getElement: AudioFactory = () => element.element as unknown as HTMLAudioElement
  return { plays: createAudioPlayer(getElement), ...element }
}

describe('a recording loaded', () => {
  it('reports the element position without a request', () => {
    const { plays, element } = player()

    plays.load(TALK)

    expect(plays.url.value).toBe(TALK)
    expect(element.loaded).toStrictEqual([TALK])
    expect(element.started).toBe(0)
    expect(plays.isPlaying.value).toBe(false)
  })

  it('loads an address once', () => {
    const { plays, element, setCurrentTime } = player()
    plays.load(TALK)
    setCurrentTime(4)

    plays.load(TALK)

    expect(element.loaded).toStrictEqual([TALK])
    expect(plays.position.value).toBe(4_000)
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
    expect(plays.position.value).toBe(0)
    expect(plays.duration.value).toBe(0)
  })
})

describe('a moment gone to', () => {
  it('loads the recording first where nothing is loaded', () => {
    const { plays, element } = player()

    plays.seek(TALK, 2_500)

    expect(element.loaded).toStrictEqual([TALK])
    expect(element.currentTime).toBe(2.5)
    expect(plays.position.value).toBe(2_500)
  })

  it('is the beginning when it is before that', () => {
    const { plays, element } = player()

    plays.seek(TALK, -400)

    expect(element.currentTime).toBe(0)
    expect(plays.position.value).toBe(0)
  })

  it('is nowhere at all where the application named no address', () => {
    const { plays, element } = player()

    plays.seek('', 2_500)

    expect(element.loaded).toStrictEqual([])
    expect(plays.position.value).toBe(0)
  })
})

describe('a recording played', () => {
  it('takes over the element from the previous recording', async () => {
    const { plays, element } = player()
    plays.play(TALK)
    await Promise.resolve()

    plays.play(OTHER)
    await Promise.resolve()

    expect(element.loaded).toStrictEqual([TALK, OTHER])
    expect(plays.url.value).toBe(OTHER)
    expect(plays.isPlaying.value).toBe(true)
  })

  it('is not playing where the window refused it, and says so', async () => {
    const { plays, element } = player()
    element.fails = true

    plays.play(TALK)
    await Promise.resolve()
    await Promise.resolve()

    expect(plays.isPlaying.value).toBe(false)
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

  it('pauses and keeps the position', () => {
    const { plays, setCurrentTime } = player()
    plays.play(TALK)
    setCurrentTime(3)

    plays.pause()

    expect(plays.isPlaying.value).toBe(false)
    expect(plays.position.value).toBe(3_000)
  })

  it('stops of itself when it ends', () => {
    const { plays, emit } = player()
    plays.play(TALK)

    emit('ended')

    expect(plays.isPlaying.value).toBe(false)
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

  it('is read again where the element learns it without a durationchange', () => {
    const { plays, element, emit, setDuration } = player()
    plays.load(TALK)
    element.readyState = 4
    setDuration(0)
    emit('loadedmetadata')
    expect(plays.duration.value).toBe(0)

    // WebKitGTK sets the estimated length of a VBR mp3 with no Xing header this way.
    element.duration = 3352.4
    emit('canplay')

    expect(plays.duration.value).toBe(3_352_400)
  })

  it('is polled while it is unknown after loadedmetadata', () => {
    vi.useFakeTimers()
    try {
      const { plays, element, setDuration } = player()
      plays.load(TALK)
      element.readyState = 1
      setDuration(0)

      element.duration = 187
      vi.advanceTimersByTime(300)

      expect(plays.duration.value).toBe(187_000)
      expect(vi.getTimerCount()).toBe(0)
    } finally {
      vi.useRealTimers()
    }
  })

  it('stops asking for the length of a recording another one replaced', () => {
    vi.useFakeTimers()
    try {
      const { plays, element, setDuration } = player()
      plays.load(TALK)
      element.readyState = 1
      setDuration(0)

      plays.load(OTHER)
      vi.advanceTimersByTime(300)

      expect(vi.getTimerCount()).toBe(0)
      expect(plays.duration.value).toBe(0)
    } finally {
      vi.useRealTimers()
    }
  })

  it('reports zero duration until the element reports one', () => {
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
    const element = createElement()
    const plays = createAudioPlayer(() => {
      made++
      return element.element as unknown as HTMLAudioElement
    })

    for (let index = 0; index < 100; index++)
      plays.load(`http://127.0.0.1:1/files/w/v/${index}.mp3`)

    expect(made).toBe(1)
    expect(element.countListeners('timeupdate')).toBe(1)
    expect(element.countListeners('error')).toBe(1)
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
    expect(plays.isPlaying.value).toBe(false)
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

describe('an unused player', () => {
  it('has no error to say', () => {
    const { plays } = player()

    expect(plays.error.value).toBe('')
  })
})

describe('the element once it is made', () => {
  it('loads only the metadata of a recording until it is played', () => {
    const { plays, element } = player()

    plays.load(TALK)

    expect(element.preload).toBe('metadata')
  })

  it('is told where a seek landed when the element says it has seeked', () => {
    const { plays, element, emit } = player()
    plays.load(TALK)

    element.currentTime = 7.5
    emit('seeked')

    expect(plays.position.value).toBe(7_500)
  })
})

describe('a recording already loaded', () => {
  it('plays from the current position', () => {
    const { plays, element } = player()
    plays.load(TALK)

    plays.play(TALK)

    expect(element.started).toBe(1)
    expect(element.loaded).toStrictEqual([TALK])
  })
})

describe('a play the window refused', () => {
  it('says the recording could not be played where it gave no reason', async () => {
    const { plays, element } = player()
    element.fails = true
    element.playError = null as unknown as Error

    plays.play(TALK)
    await Promise.resolve()
    await Promise.resolve()

    expect(plays.error.value).toBe(WORDS.unreadable)
    expect(plays.isPlaying.value).toBe(false)
  })

  it('keeps the duration the element reported', async () => {
    const { plays, element, setError } = player()
    element.fails = true

    plays.play(TALK)
    setError(2)
    await Promise.resolve()
    await Promise.resolve()

    expect(plays.error.value).toBe(WORDS.unreached)
  })
})

describe('an element that reports an error with no reason', () => {
  it('reports a failed play', () => {
    const { plays, emit } = player()
    plays.load(TALK)

    emit('error')

    expect(plays.error.value).toBe(WORDS.unreadable)
  })
})

describe('createMediaTypeProbe', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('says yes to a type an audio element of its own says it can play', () => {
    const canPlayType = vi.fn<(type: string) => CanPlayTypeResult>(() => 'maybe')
    const create = vi
      .spyOn(document, 'createElement')
      .mockReturnValue({ canPlayType } as unknown as HTMLElement)

    expect(createMediaTypeProbe()('audio/mpeg')).toBe(true)
    expect(create).toHaveBeenCalledWith('audio')
    expect(canPlayType).toHaveBeenCalledWith('audio/mpeg')
    expect(createMediaTypeProbe()('audio/vnd.wave')).toBe(true)
    expect(canPlayType).toHaveBeenCalledWith('audio/wav')
  })

  it('says no to a type the audio element answers with nothing for', () => {
    vi.spyOn(document, 'createElement').mockReturnValue({
      canPlayType: () => '',
    } as unknown as HTMLElement)

    expect(createMediaTypeProbe()('audio/mpeg')).toBe(false)
  })

  it('returns false when canPlayType throws', () => {
    vi.spyOn(document, 'createElement').mockImplementation(() => {
      throw new Error('no document')
    })

    expect(createMediaTypeProbe()('audio/mpeg')).toBe(false)
  })
})

describe('the length of a recording that does not hold the player', () => {
  /** A player whose every element is kept, the first being the one it plays through. */
  const createPlayers = () => {
    const made: ReturnType<typeof createElement>[] = []
    const plays = createAudioPlayer(() => {
      const one = createElement()
      made.push(one)
      return one.element as unknown as HTMLAudioElement
    })
    return { plays, made }
  }

  it('is read through an element of its own, and the playing recording keeps the player', async () => {
    const { plays, made } = createPlayers()
    plays.play(TALK)

    const length = plays.getDuration(OTHER)
    const reader = made[1]!
    expect(reader.element.preload).toBe('metadata')
    expect(reader.element.src).toBe(OTHER)
    reader.element.readyState = 1
    reader.setDuration(187)

    await expect(length).resolves.toBe(187_000)
    expect(plays.url.value).toBe(TALK)
    expect(made[0]!.element.loaded).toStrictEqual([TALK])
  })

  it('makes one reader element per address', async () => {
    const { plays, made } = createPlayers()

    const first = plays.getDuration(OTHER)
    made[0]!.setDuration(42)
    await first
    await plays.getDuration(OTHER)

    expect(made).toHaveLength(1)
  })

  it('is zero where the element cannot read the recording', async () => {
    const { plays, made } = createPlayers()

    const length = plays.getDuration(OTHER)
    made[0]!.setError(4)

    await expect(length).resolves.toBe(0)
  })
})
