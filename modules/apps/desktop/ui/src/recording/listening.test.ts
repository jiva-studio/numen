/**
 * One recording as its tab hears it.
 *
 * Where the player is sent, which cue is marked while it plays, and when the
 * words are asked for again are decisions, and they are the ones asked about
 * here.
 */
import { afterEach, describe, expect, it } from 'vitest'
import {
  asking,
  listening,
  playable,
  timed,
  type Cue,
  type Listened,
  type Recordings,
} from './listening'
import { WORDS } from './words'

const CUES: readonly Cue[] = [
  { text: 'The first thing said.', from: 0, to: 2_000 },
  { text: 'The second thing said.', from: 2_500, to: 5_000 },
  { text: 'The third thing said.', from: 5_000, to: 9_000 },
]

const LISTENED: Listened = {
  length: 9_000,
  heard: 9_000,
  media: 'http://127.0.0.1:1/files/w/v/talk.mp3',
  type: 'audio/mpeg',
}

/**
 * A recording of three cues, recording every question put to it. It answers
 * about a run of the words with `at`, and with nothing where no cue holds one.
 */
function talk(
  cues: readonly Cue[] | Error = CUES,
  said: Listened | Error = LISTENED,
  at: number | null = null,
) {
  /** Every address asked of it, in the order they were asked. */
  const asked: string[] = []

  const recordings: Recordings = {
    listened: async (path) => {
      asked.push(`about ${path}`)
      if (said instanceof Error) throw said
      return said
    },
    cues: async (path) => {
      asked.push(`cues ${path}`)
      if (cues instanceof Error) throw cues
      return cues
    },
    plays: async (path, run) => {
      asked.push(`plays ${path} ${run.start} ${run.length}`)
      return at
    },
  }

  return { recordings, asked }
}

/** A player that writes down every moment it was sent to. */
function played() {
  const sought: number[] = []
  return { player: { seek: (ms: number) => void sought.push(ms) }, sought }
}

/** Everything asked for has been answered and everything waiting has run. */
const settled = () => new Promise((done) => setTimeout(done, 0))

describe('a recording opened', () => {
  it('is played from the address the application serves it at', async () => {
    const { recordings } = talk()

    const heard = listening(recordings, 'talks/Ants.mp3')

    await settled()

    expect(heard.address.value).toBe(LISTENED.media)
  })

  it('asks what it is and what was heard in it, once each', async () => {
    const { recordings, asked } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3')

    await settled()

    expect(asked).toStrictEqual(['about talks/Ants.mp3', 'cues talks/Ants.mp3'])
    expect(heard.cues.value).toStrictEqual(CUES)
    expect(heard.length.value).toBe(9_000)
  })
})

describe('a recording nothing has listened to', () => {
  it('holds no words and says nothing went wrong', async () => {
    const { recordings } = talk([], { length: 0, heard: 0, media: '', type: '' })
    const heard = listening(recordings, 'talks/Ants.mp3')

    await settled()

    expect(heard.cues.value).toStrictEqual([])
    expect(heard.trouble.value).toBe('')
  })
})

describe('a build that cannot read a transcript', () => {
  it('says so, and the recording is still played', async () => {
    const { recordings } = talk(new Error('this build cannot read what a recording says'))
    const heard = listening(recordings, 'talks/Ants.mp3')

    await settled()

    expect(heard.trouble.value).toContain('cannot read what a recording says')
    expect(heard.address.value).toBe(LISTENED.media)
  })
})

describe('the cue being said', () => {
  it('is the one the player stands in', async () => {
    const { recordings } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3')
    await settled()

    heard.moved(3_000)

    expect(heard.current.value).toBe(1)
  })

  it('is the last one said while a silence stands there', async () => {
    const { recordings } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3')
    await settled()

    heard.moved(2_200)

    expect(heard.current.value).toBe(0)
  })

  it('is none before the first of them begins', async () => {
    const { recordings } = talk([{ text: 'Said late.', from: 4_000, to: 5_000 }])
    const heard = listening(recordings, 'talks/Ants.mp3')
    await settled()

    heard.moved(1_000)

    expect(heard.current.value).toBe(-1)
  })
})

describe('a moment gone to', () => {
  it('is played from, and is where the recording stands', async () => {
    const { recordings } = talk()
    const { player, sought } = played()
    const heard = listening(recordings, 'talks/Ants.mp3')
    heard.plays(player)

    heard.go(2_500)

    expect(sought).toStrictEqual([2_500])
    expect(heard.now.value).toBe(2_500)
  })

  it('is played from once the tab is drawn, where it was gone to before', async () => {
    const { recordings } = talk()
    const { player, sought } = played()
    const heard = listening(recordings, 'talks/Ants.mp3')

    heard.go(5_000)
    expect(sought).toStrictEqual([])

    heard.plays(player)
    expect(sought).toStrictEqual([5_000])
  })

  it('is the beginning when it is gone to before that', async () => {
    const { recordings } = talk()
    const { player, sought } = played()
    const heard = listening(recordings, 'talks/Ants.mp3')
    heard.plays(player)

    heard.go(-400)

    expect(sought).toStrictEqual([0])
  })
})

describe('a recording opened at a place in its words', () => {
  it('plays from the moment the first stretch was spoken at', async () => {
    const { recordings, asked } = talk(CUES, LISTENED, 2_500)
    const { player, sought } = played()
    const heard = listening(recordings, 'talks/Ants.mp3')
    heard.plays(player)

    await heard.reach({ start: 22, length: 6 })

    expect(asked).toContain('plays talks/Ants.mp3 22 6')
    expect(sought).toStrictEqual([2_500])
    expect(heard.current.value).toBe(1)
  })

  it('stands where it stands when no cue holds the stretch', async () => {
    const { recordings } = talk(CUES, LISTENED, null)
    const { player, sought } = played()
    const heard = listening(recordings, 'talks/Ants.mp3')
    heard.plays(player)

    await heard.reach({ start: 900_000, length: 6 })

    expect(sought).toStrictEqual([])
    expect(heard.trouble.value).toBe('')
  })

  it('says what it could not ask, and plays on', async () => {
    const { recordings } = talk()
    recordings.plays = async () => {
      throw new Error('the words are being written')
    }
    const heard = listening(recordings, 'talks/Ants.mp3')

    await heard.reach({ start: 22, length: 6 })

    expect(heard.trouble.value).toContain('the words are being written')
  })
})

describe('a transcript still growing', () => {
  it('is asked for again while a run is going, and once more when it stops', async () => {
    const { recordings, asked } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3')
    await settled()

    heard.ticks(true)
    await settled()
    heard.ticks(false)
    await settled()

    expect(asked.filter((one) => one.startsWith('cues')).length).toBe(3)
  })

  it('is left alone while nothing is listening to it', async () => {
    const { recordings, asked } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3')
    await settled()

    heard.ticks(false)
    heard.ticks(false)
    await settled()

    expect(asked.filter((one) => one.startsWith('cues')).length).toBe(1)
  })
})

describe('a recording tab that closes', () => {
  it('holds no words, and is played from no other moment', async () => {
    const { recordings } = talk()
    const { player, sought } = played()
    const heard = listening(recordings, 'talks/Ants.mp3')
    heard.plays(player)
    await settled()

    heard.close()
    heard.go(4_000)

    expect(heard.cues.value).toStrictEqual([])
    expect(sought).toStrictEqual([])
  })
})

describe('a moment written out', () => {
  it('is read as a clock, and carries an hour only where there is one', () => {
    expect(timed(0)).toBe('0:00')
    expect(timed(9_400)).toBe('0:09')
    expect(timed(125_000)).toBe('2:05')
    expect(timed(3_725_000)).toBe('1:02:05')
  })
})

describe('what this window can play', () => {
  afterEach(() => asking(() => true))

  it('asks the window once, however many recordings are open', () => {
    let asks = 0
    asking((type) => {
      asks++
      return type === 'audio/mpeg'
    })

    expect(playable('audio/mpeg')).toBe(true)
    expect(playable('audio/mpeg')).toBe(true)
    expect(playable('audio/mpeg')).toBe(true)
    expect(asks).toBe(1)

    expect(playable('audio/wav')).toBe(false)
    expect(asks).toBe(2)
  })

  it('plays nothing where the application named no type', () => {
    asking(() => true)
    expect(playable('')).toBe(false)
  })

  it('plays nothing where the window answers for nothing', () => {
    asking(() => false)
    expect(playable('audio/mpeg')).toBe(false)
  })
})

describe('a transcript asked for twice at once', () => {
  it('keeps the answer to the later asking, however they arrive', async () => {
    // The first asking is answered with fewer words, and answered last.
    const early: readonly Cue[] = [CUES[0]!]
    const first: { answer: ((cues: readonly Cue[]) => void) | null } = { answer: null }

    const recordings: Recordings = {
      listened: async () => LISTENED,
      cues: () =>
        first.answer
          ? Promise.resolve(CUES)
          : new Promise<readonly Cue[]>((done) => {
              first.answer = done
            }),
      plays: async () => null,
    }

    const heard = listening(recordings, 'talks/Ants.mp3')
    heard.ticks(true)
    await settled()

    // The later asking lands first, then the earlier one answers.
    first.answer?.(early)
    await settled()

    expect(heard.cues.value).toStrictEqual(CUES)
  })
})

describe('a player that could not load the recording', () => {
  it('says which of the failures it was', async () => {
    const said: Record<number, string> = {}
    for (const code of [1, 2, 3, 4]) {
      const { recordings } = talk()
      const heard = listening(recordings, 'talks/Ants.mp3')
      heard.failed(code)
      said[code] = heard.broken.value
    }

    expect(new Set(Object.values(said)).size).toBe(4)
    expect(said[2]).toContain('read from the vault')
    expect(said[3]).toContain('cannot decode')
    expect(said[4]).toContain('does not play recordings of this kind')
  })

  it('says something for a failure it has no name for', async () => {
    const { recordings } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3')

    heard.failed(undefined)

    expect(heard.broken.value).not.toBe('')
  })
})

describe('what the tab says where the words would stand', () => {
  it('is what went wrong before it is anything else', async () => {
    const { recordings } = talk(new Error('the words are being written'))
    const heard = listening(recordings, 'talks/Ants.mp3')
    await settled()

    heard.ticks(true)

    expect(heard.note.value).toContain('the words are being written')
  })

  it('is that a run is going, where nothing went wrong', async () => {
    const { recordings } = talk([])
    const heard = listening(recordings, 'talks/Ants.mp3')
    await settled()

    heard.ticks(true)

    expect(heard.note.value).toBe(WORDS.transcribing)
  })

  it('is nothing at all once there are words', async () => {
    const { recordings } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3')
    await settled()

    expect(heard.note.value).toBe('')
  })
})
