/**
 * One recording as its tab hears it.
 *
 * Where the player is sent, which cue is marked while it plays, and when the
 * words are asked for again are decisions, and they are the ones asked about
 * here.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { ref } from 'vue'
import {
  asking,
  listening,
  plays,
  timed,
  type Cue,
  type Listened,
  type Recordings,
  type Spoken,
} from './listening'
import type { Player } from './playing'
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

  /** Every set of words written to it, in the order they were written. */
  const written: (readonly Cue[])[] = []

  const recordings: Recordings = {
    listened: async (path) => {
      asked.push(`about ${path}`)
      if (said instanceof Error) throw said
      return said
    },
    cues: async (path) => {
      asked.push(`cues ${path}`)
      if (cues instanceof Error) throw cues
      return { cues, editable: true }
    },
    writes: async (path, kept) => {
      asked.push(`writes ${path}`)
      written.push(kept)
    },
    plays: async (path, run) => {
      asked.push(`plays ${path} ${run.start} ${run.length}`)
      return at
    },
  }

  return { recordings, asked, written }
}

/**
 * The one player the window has, faked: it writes down every moment it was
 * sent to and holds one recording at a time, as the real one does.
 */
function played() {
  const address = ref('')
  const at = ref(0)
  const length = ref(0)
  const playing = ref(false)
  const failed = ref('')
  const sought: number[] = []

  const player: Player = {
    address,
    at,
    length,
    playing,
    failed,
    load: (wanted) => void (address.value = wanted),
    play: (wanted) => {
      address.value = wanted
      playing.value = true
    },
    pause: () => void (playing.value = false),
    seek: (wanted, ms) => {
      address.value = wanted
      at.value = ms
      sought.push(ms)
    },
  }

  /** The recording plays on: the player moves of itself. */
  const moves = (ms: number) => {
    address.value = LISTENED.media
    at.value = ms
  }

  return { player, sought, moves, failed, address }
}

/** Everything asked for has been answered and everything waiting has run. */
const settled = () => new Promise((done) => setTimeout(done, 0))

/** The words have been still long enough to be written, and were. */
const still = () => new Promise((done) => setTimeout(done, 20))

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
    const { player, moves } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)
    await settled()

    moves(3_000)

    expect(heard.current.value).toBe(1)
  })

  it('is the last one said while a silence stands there', async () => {
    const { recordings } = talk()
    const { player, moves } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)
    await settled()

    moves(2_200)

    expect(heard.current.value).toBe(0)
  })

  it('is none before the first of them begins', async () => {
    const { recordings } = talk([{ text: 'Said late.', from: 4_000, to: 5_000 }])
    const { player, moves } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)
    await settled()

    moves(1_000)

    expect(heard.current.value).toBe(-1)
  })

  it('is none while the player holds another recording', async () => {
    const { recordings } = talk()
    const { player, moves } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)
    await settled()

    moves(3_000)
    expect(heard.now.value).toBe(3_000)

    // The player is given another recording: this one stands at its beginning
    // until somebody plays it again.
    player.load('http://127.0.0.1:1/files/w/v/another.mp3')

    expect(heard.now.value).toBe(0)
  })
})

describe('a moment gone to', () => {
  it('is played from, and is where the recording stands', async () => {
    const { recordings } = talk()
    const { player, sought } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)
    await settled()

    heard.go(2_500)

    expect(sought).toStrictEqual([2_500])
    expect(heard.now.value).toBe(2_500)
  })

  it('is played from once the recording knows where its bytes are', async () => {
    const { recordings } = talk()
    const { player, sought } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)

    heard.go(5_000)
    expect(sought).toStrictEqual([])

    await settled()
    expect(sought).toStrictEqual([5_000])
  })

  it('is the beginning when it is gone to before that', async () => {
    const { recordings } = talk()
    const { player, sought } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)
    await settled()

    heard.go(-400)

    expect(sought).toStrictEqual([0])
  })
})

describe('the one player the window has', () => {
  it('is taken by whichever recording is played', async () => {
    const { recordings } = talk()
    const { player } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)
    await settled()

    heard.play()

    expect(player.address.value).toBe(LISTENED.media)
    expect(heard.playing.value).toBe(true)
  })

  it('says this recording is not playing while it holds another', async () => {
    const { recordings } = talk()
    const { player } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)
    await settled()

    heard.play()
    player.play('http://127.0.0.1:1/files/w/v/another.mp3')

    expect(heard.playing.value).toBe(false)
  })

  it('is stopped only by the recording it holds', async () => {
    const { recordings } = talk()
    const { player } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)
    await settled()

    player.play('http://127.0.0.1:1/files/w/v/another.mp3')
    heard.pause()

    expect(player.playing.value).toBe(true)
  })
})

describe('a recording opened at a place in its words', () => {
  it('plays from the moment the first stretch was spoken at', async () => {
    const { recordings, asked } = talk(CUES, LISTENED, 2_500)
    const { player, sought } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)

    await heard.reach({ start: 22, length: 6 })

    expect(asked).toContain('plays talks/Ants.mp3 22 6')
    expect(sought).toStrictEqual([2_500])
    expect(heard.current.value).toBe(1)
  })

  it('stands where it stands when no cue holds the stretch', async () => {
    const { recordings } = talk(CUES, LISTENED, null)
    const { player, sought } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)

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
    const heard = listening(recordings, 'talks/Ants.mp3', player)
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

    expect(plays('audio/mpeg')).toBe(true)
    expect(plays('audio/mpeg')).toBe(true)
    expect(plays('audio/mpeg')).toBe(true)
    expect(asks).toBe(1)

    expect(plays('audio/wav')).toBe(false)
    expect(asks).toBe(2)
  })

  it('plays nothing where the application named no type', () => {
    asking(() => true)
    expect(plays('')).toBe(false)
  })

  it('plays nothing where the window answers for nothing', () => {
    asking(() => false)
    expect(plays('audio/mpeg')).toBe(false)
  })
})

describe('a transcript asked for twice at once', () => {
  it('keeps the answer to the later asking, however they arrive', async () => {
    // The first asking is answered with fewer words, and answered last.
    const early: Spoken = { cues: [CUES[0]!], editable: true }
    const first: { answer: ((said: Spoken) => void) | null } = { answer: null }

    const recordings: Recordings = {
      listened: async () => LISTENED,
      cues: () =>
        first.answer
          ? Promise.resolve({ cues: CUES, editable: true })
          : new Promise<Spoken>((done) => {
              first.answer = done
            }),
      writes: async () => {},
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
  it('says so under the recording it could not load', async () => {
    const { recordings } = talk()
    const { player, failed, address } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)
    await settled()

    address.value = LISTENED.media
    failed.value = 'the recording could not be read from the vault'

    expect(heard.broken.value).toContain('read from the vault')
  })

  it('says nothing where the failure was another recording', async () => {
    const { recordings } = talk()
    const { player, failed, address } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)
    await settled()

    address.value = 'http://127.0.0.1:1/files/w/v/another.mp3'
    failed.value = 'the recording could not be read from the vault'

    expect(heard.broken.value).toBe('')
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

describe('the words as a person edits them', () => {
  it('stand in the editor, one cue to a line', async () => {
    const { recordings } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3')
    await settled()

    expect(heard.prose.value).toBe(
      'The first thing said.\nThe second thing said.\nThe third thing said.',
    )
  })

  it('are written once they have been still', async () => {
    const { recordings, written } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3', played().player, 5)
    await settled()

    heard.typed('The first thing said.\nThe second thing heard.\nThe third thing said.')
    expect(written).toStrictEqual([])

    await still()

    expect(written).toStrictEqual([
      [
        CUES[0],
        { text: 'The second thing heard.', from: 2_500, to: 5_000 },
        CUES[2],
      ],
    ])
  })

  it('are written once for a run of typing', async () => {
    const { recordings, written } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3', played().player, 5)
    await settled()

    heard.typed('The first thing said.\nThe second thing h\nThe third thing said.')
    heard.typed('The first thing said.\nThe second thing he\nThe third thing said.')
    heard.typed('The first thing said.\nThe second thing heard.\nThe third thing said.')
    await still()

    expect(written.length).toBe(1)
  })

  it('are the cues the tab then holds', async () => {
    const { recordings } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3', played().player, 5)
    await settled()

    heard.typed('The first thing said.\n\nThe third thing said.')
    await still()

    expect(heard.cues.value).toStrictEqual([CUES[0], CUES[2]])
  })

  it('stay on screen while a transcript arriving beside them is read', async () => {
    const { recordings } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3', played().player, 200)
    await settled()

    heard.typed('Mine.\nThe second thing said.\nThe third thing said.')
    heard.ticks(true)
    await settled()

    expect(heard.prose.value.startsWith('Mine.')).toBe(true)
  })

  it('are owed again where the write was refused', async () => {
    const { recordings } = talk()
    recordings.writes = async () => {
      throw new Error('the transcript is held')
    }
    const heard = listening(recordings, 'talks/Ants.mp3', played().player, 5)
    await settled()

    heard.typed('Mine.\nThe second thing said.\nThe third thing said.')
    await still()

    expect(heard.trouble.value).toContain('the transcript is held')
    expect(heard.cues.value).toStrictEqual(CUES)
  })

  it('reach the file as the tab closes', async () => {
    const { recordings, written } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3', played().player, 10_000)
    await settled()

    heard.typed('Mine.\nThe second thing said.\nThe third thing said.')
    heard.close()
    await settled()

    expect(written.length).toBe(1)
  })
})

describe('a transcript a run still holds', () => {
  it('is not edited', async () => {
    const recordings: Recordings = {
      listened: async () => LISTENED,
      cues: async () => ({ cues: CUES, editable: false }),
      writes: async () => {},
      plays: async () => null,
    }
    const heard = listening(recordings, 'talks/Ants.mp3')
    await settled()

    expect(heard.editable.value).toBe(false)
  })

  it('is edited once nothing holds it', async () => {
    const { recordings } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3')
    await settled()

    expect(heard.editable.value).toBe(true)
  })
})

describe('the line a person asked for', () => {
  it('is played from the moment it was said at', async () => {
    const { recordings } = talk()
    const { player, sought } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)
    await settled()

    heard.goes(2)

    expect(sought).toStrictEqual([5_000])
  })

  it('leaves the player where it stands where there is no such line', async () => {
    const { recordings } = talk()
    const { player, sought } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player)
    await settled()

    heard.goes(9)

    expect(sought).toStrictEqual([])
  })
})

describe('following the line being said', () => {
  it('is on as a recording opens, and is turned off and on again', async () => {
    const { recordings } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3')
    await settled()

    expect(heard.following.value).toBe(true)

    heard.follows(false)
    expect(heard.following.value).toBe(false)

    heard.follows(true)
    expect(heard.following.value).toBe(true)
  })
})

describe('the line being said while the words are edited', () => {
  it('is the one the player stands in, as the lines now read', async () => {
    const { recordings } = talk()
    const { player, moves } = played()
    const heard = listening(recordings, 'talks/Ants.mp3', player, 10_000)
    await settled()

    heard.typed('The first thing said. The second thing said.\nThe third thing said.')
    moves(3_000)

    expect(heard.current.value).toBe(0)
    expect(heard.lines.value.length).toBe(2)
  })
})

describe('typing that lands while a write is in the air', () => {
  it('is written once the one before it has answered', async () => {
    const held: { answer: (() => void) | null } = { answer: null }
    const { recordings, written } = talk()
    recordings.writes = (_, kept) => {
      written.push(kept)
      return new Promise<void>((done) => {
        held.answer = done
      })
    }
    const heard = listening(recordings, 'talks/Ants.mp3', played().player, 5)
    await settled()

    heard.typed('One.\nThe second thing said.\nThe third thing said.')
    await still()
    expect(written.length).toBe(1)

    heard.typed('Two.\nThe second thing said.\nThe third thing said.')
    held.answer?.()
    await still()

    expect(written.length).toBe(2)
    expect(written[1]![0]!.text).toBe('Two.')
  })
})

describe('a transcript nobody edited', () => {
  it('is not written down when the editor hands back what it was given', async () => {
    const { recordings, written } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3', played().player, 5)
    await settled()

    // The editor hands the document back carrying a newline of its own.
    heard.typed(heard.prose.value + '\n')
    await still()

    expect(written).toStrictEqual([])
  })

  it('is written down once a word actually changes', async () => {
    const { recordings, written } = talk()
    const heard = listening(recordings, 'talks/Ants.mp3', played().player, 5)
    await settled()

    heard.typed('The first thing Rupa said.\nThe second thing said.\nThe third thing said.')
    await still()

    expect(written).toHaveLength(1)
    expect(written[0]![0]!.text).toBe('The first thing Rupa said.')
    expect(written[0]![0]!.from).toBe(0)
  })
})
