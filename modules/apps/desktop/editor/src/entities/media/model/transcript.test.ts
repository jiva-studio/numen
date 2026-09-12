/**
 * One recording as its tab reads it.
 *
 * Where the player is sent, which cue is marked while it plays, and when the
 * words are asked for again are decisions, and they are the ones asked about
 * here.
 */
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'
import { useTranscript } from './transcript'
import type { Recordings, RecordingSummary, Transcript } from '../types'
import type { ArtifactStates } from '@/shared/artifacts'
import type { Cue } from '../lib/cues'
import { createMediaTypeProbe, type Player } from './player'
import { WORDS } from '../words'

const CUES: readonly Cue[] = [
  { text: 'The first thing said.', from: 0, to: 2_000 },
  { text: 'The second thing said.', from: 2_500, to: 5_000 },
  { text: 'The third thing said.', from: 5_000, to: 9_000 },
]

const SUMMARY: RecordingSummary = {
  duration: 9_000,
  mediaUrl: 'http://127.0.0.1:1/files/w/v/talk.mp3',
  mediaType: 'audio/mpeg',
  url: '',
}

/**
 * A recording of three cues, recording every question put to it. It answers
 * about a run of the words with `at`, and with nothing where no cue holds one.
 */
function talk(
  cues: readonly Cue[] | Error = CUES,
  summary: RecordingSummary | Error = SUMMARY,
  at: number | null = null,
  states: ArtifactStates = { transcript: 'done' },
) {
  /** Every address asked of it, in the order they were asked. */
  const asked: string[] = []

  /** Every set of words written to it, in the order they were written. */
  const written: (readonly Cue[])[] = []

  const recordings: Recordings = {
    getSummary: async (path) => {
      asked.push(`about ${path}`)
      if (summary instanceof Error) throw summary
      return summary
    },
    getTaskStates: async (path) => {
      asked.push(`carries ${path}`)
      return states
    },
    readTranscript: async (path) => {
      asked.push(`cues ${path}`)
      if (cues instanceof Error) throw cues
      return { cues, isEditable: true, prose: '' }
    },
    readArticle: async (path) => {
      asked.push(`prose ${path}`)
      return { cues: [], isEditable: true, prose: 'A page nothing timed.' }
    },
    writeTranscript: async (path, kept) => {
      asked.push(`writes ${path}`)
      written.push(kept)
    },
    findCueTime: async (path, run) => {
      asked.push(`plays ${path} ${run.from} ${run.to}`)
      return at
    },
  }

  return { recordings, asked, written }
}

/**
 * The one player the window has, faked: it writes down every moment it was
 * sent to and holds one recording at a time, as the real one does.
 */
function createPlayer() {
  const address = ref('')
  const at = ref(0)
  const duration = ref(0)
  const playing = ref(false)
  const error = ref('')
  const sought: number[] = []

  const player: Player = {
    url: address,
    at,
    duration,
    playing,
    error,
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
  const setCurrentTime = (ms: number) => {
    address.value = SUMMARY.mediaUrl
    at.value = ms
  }

  return { player, sought, setCurrentTime, error, address, duration, playing, at }
}

/** Everything asked for has been answered and everything waiting has run. */
const flush = () => new Promise((done) => setTimeout(done, 0))

/** The words have been still long enough to be written, and were. */
const still = () => new Promise((done) => setTimeout(done, 20))

describe('a recording opened', () => {
  it('is played from the address the application serves it at', async () => {
    const { recordings } = talk()

    const heard = useTranscript(recordings, 'talks/Ants.mp3')

    await flush()

    expect(heard.url.value).toBe(SUMMARY.mediaUrl)
  })

  it('asks what it is, what it carries and what was heard in it, once each', async () => {
    const { recordings, asked } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3')

    await flush()

    expect(asked).toStrictEqual([
      'about talks/Ants.mp3',
      'carries talks/Ants.mp3',
      'cues talks/Ants.mp3',
    ])
    expect(heard.cues.value).toStrictEqual(CUES)
    expect(heard.duration.value).toBe(9_000)
  })
})

describe('a recording nothing has listened to', () => {
  it('holds no words and says nothing went wrong', async () => {
    const { recordings } = talk([], { duration: 0, mediaUrl: '', mediaType: '', url: '' })
    const heard = useTranscript(recordings, 'talks/Ants.mp3')

    await flush()

    expect(heard.cues.value).toStrictEqual([])
    expect(heard.error.value).toBe('')
  })
})

describe('a build that cannot read a transcript', () => {
  it('says so, and the recording is still played', async () => {
    const { recordings } = talk(new Error('this build cannot read what a recording says'))
    const heard = useTranscript(recordings, 'talks/Ants.mp3')

    await flush()

    expect(heard.error.value).toContain('numen did not answer')
    expect(heard.url.value).toBe(SUMMARY.mediaUrl)
  })
})

describe('the cue being said', () => {
  it('is the one the player stands in', async () => {
    const { recordings } = talk()
    const { player, setCurrentTime } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    setCurrentTime(3_000)

    expect(heard.current.value).toBe(1)
  })

  it('is the last one said while a silence stands there', async () => {
    const { recordings } = talk()
    const { player, setCurrentTime } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    setCurrentTime(2_200)

    expect(heard.current.value).toBe(0)
  })

  it('is none before the first of them begins', async () => {
    const { recordings } = talk([{ text: 'Said late.', from: 4_000, to: 5_000 }])
    const { player, setCurrentTime } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    setCurrentTime(1_000)

    expect(heard.current.value).toBe(-1)
  })

  it('is none while the player holds another recording', async () => {
    const { recordings } = talk()
    const { player, setCurrentTime } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    setCurrentTime(3_000)
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
    const { player, sought } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    heard.go(2_500)

    expect(sought).toStrictEqual([2_500])
    expect(heard.now.value).toBe(2_500)
  })

  it('is played from once the recording knows where its bytes are', async () => {
    const { recordings } = talk()
    const { player, sought } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })

    heard.go(5_000)
    expect(sought).toStrictEqual([])

    await flush()
    expect(sought).toStrictEqual([5_000])
  })

  it('is the beginning when it is gone to before that', async () => {
    const { recordings } = talk()
    const { player, sought } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    heard.go(-400)

    expect(sought).toStrictEqual([0])
  })
})

describe('the one player the window has', () => {
  it('is taken by whichever recording is played', async () => {
    const { recordings } = talk()
    const { player } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    heard.play()

    expect(player.url.value).toBe(SUMMARY.mediaUrl)
    expect(heard.playing.value).toBe(true)
  })

  it('says this recording is not playing while it holds another', async () => {
    const { recordings } = talk()
    const { player } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    heard.play()
    player.play('http://127.0.0.1:1/files/w/v/another.mp3')

    expect(heard.playing.value).toBe(false)
  })

  it('is stopped only by the recording it holds', async () => {
    const { recordings } = talk()
    const { player } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    player.play('http://127.0.0.1:1/files/w/v/another.mp3')
    heard.pause()

    expect(player.playing.value).toBe(true)
  })
})

describe('a file whose text the vault carries under one kind and not the other', () => {
  it('reads the words with their times where the file carries a transcript', async () => {
    const { recordings, asked } = talk(CUES, SUMMARY, null, { transcript: 'done' })

    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    expect(asked).toContain('cues talks/Ants.mp3')
    expect(asked).not.toContain('prose talks/Ants.mp3')
    expect(heard.cues.value).toStrictEqual(CUES)
  })

  it('reads the prose where the file carries an article', async () => {
    const { recordings, asked } = talk(CUES, SUMMARY, null, { article: 'done' })

    const heard = useTranscript(recordings, 'notes/A page.md')
    await flush()

    expect(asked).toContain('prose notes/A page.md')
    expect(asked).not.toContain('cues notes/A page.md')
    expect(heard.cues.value).toStrictEqual([])
    expect(heard.prose.value).toBe('A page nothing timed.')
  })
})

describe('a recording opened at a place in its words', () => {
  it('plays from the moment the first span was spoken at', async () => {
    const { recordings, asked } = talk(CUES, SUMMARY, 2_500)
    const { player, sought } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })

    await heard.reach({ from: 22, to: 28 })

    expect(asked).toContain('plays talks/Ants.mp3 22 28')
    expect(sought).toStrictEqual([2_500])
    expect(heard.current.value).toBe(1)
  })

  it('stands where it stands when no cue holds the span', async () => {
    const { recordings } = talk(CUES, SUMMARY, null)
    const { player, sought } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })

    await heard.reach({ from: 900_000, to: 900_006 })

    expect(sought).toStrictEqual([])
    expect(heard.error.value).toBe('')
  })

  it('says what it could not ask, and plays on', async () => {
    const { recordings } = talk()
    recordings.findCueTime = async () => {
      throw new Error('the words are being written')
    }
    const heard = useTranscript(recordings, 'talks/Ants.mp3')

    await heard.reach({ from: 22, to: 28 })

    expect(heard.error.value).toContain('numen did not answer')
    expect(heard.error.value).not.toContain('the words are being written')
  })
})

describe('a transcript still growing', () => {
  it('is asked for again while a run is going, and once more when it stops', async () => {
    const { recordings, asked } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    heard.setWorking(true)
    await flush()
    heard.setWorking(false)
    await flush()

    expect(asked.filter((one) => one.startsWith('cues')).length).toBe(3)
  })

  it('is left alone while nothing is listening to it', async () => {
    const { recordings, asked } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    heard.setWorking(false)
    heard.setWorking(false)
    await flush()

    expect(asked.filter((one) => one.startsWith('cues')).length).toBe(1)
  })
})

describe('two questions about the words in flight at once', () => {
  /**
   * The words answered on the first asking only, and held back until the test
   * lets them go. Everything asked after that never comes back.
   */
  const slow = () => {
    let asks = 0
    let letGo: (spoken: Transcript) => void = () => {}
    const held = new Promise<Transcript>((done) => {
      letGo = done
    })
    const recordings: Recordings = {
      getSummary: async () => (++asks > 1 ? new Promise<RecordingSummary>(() => {}) : SUMMARY),
      getTaskStates: async () => ({ transcript: 'done' }),
      readTranscript: async () => held,
      readArticle: async () => held,
      writeTranscript: async () => {},
      findCueTime: async () => null,
    }
    return { recordings, letGo: () => letGo({ cues: CUES, isEditable: true, prose: '' }) }
  }

  // The older answer carries less than the newer, so it is drawn only while
  // nothing newer has been.
  it('draws the older answer where the newer has not come back', async () => {
    const { recordings, letGo } = slow()
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    heard.setWorking(true)
    await flush()

    letGo()
    await flush()

    expect(heard.cues.value).toStrictEqual(CUES)
  })
})

describe('a recording tab that closes', () => {
  it('holds no words, and is played from no other moment', async () => {
    const { recordings } = talk()
    const { player, sought } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    heard.close()
    heard.go(4_000)

    expect(heard.cues.value).toStrictEqual([])
    expect(sought).toStrictEqual([])
  })

  it('stops the recording it holds', async () => {
    const { recordings } = talk()
    const { player } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()
    heard.play()
    expect(player.playing.value).toBe(true)

    heard.close()

    expect(player.playing.value).toBe(false)
  })

  it('leaves another recording playing', async () => {
    const { recordings } = talk()
    const { player } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()
    player.play('http://127.0.0.1:1/files/w/v/another.mp3')

    heard.close()

    expect(player.playing.value).toBe(true)
    expect(player.url.value).toBe('http://127.0.0.1:1/files/w/v/another.mp3')
  })

  it('leaves the recording where it stopped, for whoever opens it again', async () => {
    const { recordings } = talk()
    const { player, at } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()
    heard.play()
    at.value = 6_200

    heard.close()
    const again = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    expect(player.url.value).toBe(SUMMARY.mediaUrl)
    expect(again.now.value).toBe(6_200)
  })
})

describe('a recording tab as it opens', () => {
  it('puts its recording in a player holding nothing, and plays none of it', async () => {
    const { recordings } = talk()
    const { player } = createPlayer()

    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    expect(player.url.value).toBe(SUMMARY.mediaUrl)
    expect(player.playing.value).toBe(false)
    expect(heard.playing.value).toBe(false)
  })

  it('leaves the player alone where another recording is playing', async () => {
    const { recordings } = talk()
    const { player } = createPlayer()
    player.play('http://127.0.0.1:1/files/w/v/another.mp3')

    useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    expect(player.url.value).toBe('http://127.0.0.1:1/files/w/v/another.mp3')
    expect(player.playing.value).toBe(true)
  })
})

describe('how long the recording runs, as the controls read it', () => {
  it('is what the application said about it', async () => {
    const { recordings } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player })
    await flush()

    expect(heard.runs.value).toBe(9_000)
  })

  it('is what the recording itself says, where nothing has listened to it', async () => {
    const { recordings } = talk([], { ...SUMMARY, duration: 0 })
    const { player, duration } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    duration.value = 85_000

    expect(heard.runs.value).toBe(85_000)
  })

  it('is what the application said while the player holds another recording', async () => {
    const { recordings } = talk()
    const { player, duration } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    player.load('http://127.0.0.1:1/files/w/v/another.mp3')
    duration.value = 400_000

    expect(heard.runs.value).toBe(9_000)
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

describe('a transcript asked for twice at once', () => {
  it('keeps the answer to the later asking, however they arrive', async () => {
    // The earlier asking is answered with fewer words, and answered last.
    const early: Transcript = { cues: [CUES[0]!], isEditable: true, prose: '' }
    const answers: ((said: Transcript) => void)[] = []

    const recordings: Recordings = {
      getSummary: async () => SUMMARY,
      getTaskStates: async () => ({ transcript: 'done' }),
      readTranscript: () => new Promise<Transcript>((done) => void answers.push(done)),
      readArticle: async () => early,
      writeTranscript: async () => {},
      findCueTime: async () => null,
    }

    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()
    heard.setWorking(true)
    await flush()

    // The later asking lands first, then the earlier one answers.
    answers[1]?.({ cues: CUES, isEditable: true, prose: '' })
    await flush()
    answers[0]?.(early)
    await flush()

    expect(heard.cues.value).toStrictEqual(CUES)
  })
})

describe('a player that could not load the recording', () => {
  it('says so under the recording it could not load', async () => {
    const { recordings } = talk()
    const { player, error, address } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    address.value = SUMMARY.mediaUrl
    error.value = 'the recording could not be read from the vault'

    expect(heard.broken.value).toContain('read from the vault')
  })

  it('says nothing where the failure was another recording', async () => {
    const { recordings } = talk()
    const { player, error, address } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    address.value = 'http://127.0.0.1:1/files/w/v/another.mp3'
    error.value = 'the recording could not be read from the vault'

    expect(heard.broken.value).toBe('')
  })
})

describe('what the tab says where the words would stand', () => {
  // What went wrong is drawn above the words, so it is read whether or not
  // there are any and the note goes on saying what the transcript is.
  it('is what the transcript is, and what went wrong is said apart from it', async () => {
    const { recordings } = talk(new Error('the words are being written'))
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    heard.setWorking(true)

    expect(heard.error.value).toContain('numen did not answer')
    expect(heard.note.value).toBe(WORDS.transcribing)
  })

  it('says what went wrong even where the words are on screen', async () => {
    const { recordings } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player, quiet: 5 })
    await flush()

    recordings.writeTranscript = async () => {
      throw new Error('the transcript is being listened to')
    }
    heard.setProse('One.\nThe second thing said.\nThe third thing said.')
    await still()

    expect(heard.note.value).toBe('')
    expect(heard.error.value).toContain('numen did not answer')
    expect(heard.error.value).not.toContain('being listened to')
  })

  it('is that a run is going, where nothing went wrong', async () => {
    const { recordings } = talk([])
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    heard.setWorking(true)

    expect(heard.note.value).toBe(WORDS.transcribing)
  })

  it('is nothing at all once there are words', async () => {
    const { recordings } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    expect(heard.note.value).toBe('')
  })

  it('is nothing at all while a run goes on adding to words already there', async () => {
    const { recordings } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    heard.setWorking(true)

    expect(heard.note.value).toBe('')
  })

  it('is that nothing was heard where the recording holds no words', async () => {
    const { recordings } = talk([])
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    expect(heard.note.value).toBe(WORDS.silence)
  })
})

describe('the moments in the editor gutter', () => {
  it('are one to a line, on a clock', async () => {
    const { recordings } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    expect(heard.times.value).toStrictEqual(['0:00', '0:02', '0:05'])
  })

  it('are none where nothing was heard in the recording and nothing was typed', async () => {
    const { recordings } = talk([])
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    expect(heard.times.value).toStrictEqual([])
    expect(heard.current.value).toBe(-1)
  })

  it('are the one line a person typed into a recording nothing was heard in', async () => {
    const { recordings } = talk([])
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player, quiet: 10_000 })
    await flush()

    heard.setProse('The first thing I heard.')

    expect(heard.times.value).toStrictEqual(['0:00'])
    expect(heard.note.value).toBe('')
  })

  // They are worked out once for the whole transcript, however long it is, and
  // the recording playing does not touch them.
  it('are not worked out again as the recording plays', async () => {
    const { recordings } = talk()
    const { player, setCurrentTime } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()
    const written = heard.times.value

    setCurrentTime(3_000)
    setCurrentTime(6_000)

    expect(heard.current.value).toBe(2)
    expect(heard.times.value).toBe(written)
  })
})

describe('the words as a person edits them', () => {
  it('stand in the editor, one cue to a line', async () => {
    const { recordings } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    expect(heard.prose.value).toBe(
      'The first thing said.\nThe second thing said.\nThe third thing said.',
    )
  })

  it('are written once they have been still', async () => {
    const { recordings, written } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player, quiet: 5 })
    await flush()

    heard.setProse('The first thing said.\nThe second thing heard.\nThe third thing said.')
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
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player, quiet: 5 })
    await flush()

    heard.setProse('The first thing said.\nThe second thing h\nThe third thing said.')
    heard.setProse('The first thing said.\nThe second thing he\nThe third thing said.')
    heard.setProse('The first thing said.\nThe second thing heard.\nThe third thing said.')
    await still()

    expect(written.length).toBe(1)
  })

  it('are the cues the tab then holds', async () => {
    const { recordings } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player, quiet: 5 })
    await flush()

    heard.setProse('The first thing said.\n\nThe third thing said.')
    await still()

    expect(heard.cues.value).toStrictEqual([CUES[0], CUES[2]])
  })

  it('stay on screen while a transcript arriving beside them is read', async () => {
    const { recordings } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player, quiet: 200 })
    await flush()

    heard.setProse('Mine.\nThe second thing said.\nThe third thing said.')
    heard.setWorking(true)
    await flush()

    expect(heard.prose.value.startsWith('Mine.')).toBe(true)
  })

  it('are owed again where the write was refused', async () => {
    const { recordings } = talk()
    recordings.writeTranscript = async () => {
      throw new Error('the transcript is held')
    }
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player, quiet: 5 })
    await flush()

    heard.setProse('Mine.\nThe second thing said.\nThe third thing said.')
    await still()

    expect(heard.error.value).toContain('numen did not answer')
    expect(heard.cues.value).toStrictEqual(CUES)
  })

  it('reach the file as the tab closes', async () => {
    const { recordings, written } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player, quiet: 10_000 })
    await flush()

    heard.setProse('Mine.\nThe second thing said.\nThe third thing said.')
    heard.close()
    await flush()

    expect(written.length).toBe(1)
  })
})

describe('a transcript a run still holds', () => {
  it('is not edited', async () => {
    const recordings: Recordings = {
      getSummary: async () => SUMMARY,
      getTaskStates: async () => ({ transcript: 'done' }),
      readTranscript: async () => ({ cues: CUES, isEditable: false, prose: '' }),
      readArticle: async () => ({ cues: [], isEditable: false, prose: '' }),
      writeTranscript: async () => {},
      findCueTime: async () => null,
    }
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    expect(heard.isEditable.value).toBe(false)
  })

  it('is edited once nothing holds it', async () => {
    const { recordings } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    expect(heard.isEditable.value).toBe(true)
  })
})

describe('the line a person asked for', () => {
  it('is played from the moment it was said at', async () => {
    const { recordings } = talk()
    const { player, sought } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    heard.goToLine(2)

    expect(sought).toStrictEqual([5_000])
  })

  it('leaves the player where it stands where there is no such line', async () => {
    const { recordings } = talk()
    const { player, sought } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player })
    await flush()

    heard.goToLine(9)

    expect(sought).toStrictEqual([])
  })
})

describe('following the line being said', () => {
  it('is on as a recording opens, and is turned off and on again', async () => {
    const { recordings } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3')
    await flush()

    expect(heard.following.value).toBe(true)

    heard.setFollowing(false)
    expect(heard.following.value).toBe(false)

    heard.setFollowing(true)
    expect(heard.following.value).toBe(true)
  })
})

describe('the line being said while the words are edited', () => {
  it('is the one the player stands in, as the lines now read', async () => {
    const { recordings } = talk()
    const { player, setCurrentTime } = createPlayer()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: player, quiet: 10_000 })
    await flush()

    heard.setProse('The first thing said. The second thing said.\nThe third thing said.')
    setCurrentTime(3_000)

    expect(heard.current.value).toBe(0)
    expect(heard.times.value.length).toBe(2)
  })
})

describe('typing that lands while a write is in the air', () => {
  it('is written once the one before it has answered', async () => {
    const held: { answer: (() => void) | null } = { answer: null }
    const { recordings, written } = talk()
    recordings.writeTranscript = (_, kept) => {
      written.push(kept)
      return new Promise<void>((done) => {
        held.answer = done
      })
    }
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player, quiet: 5 })
    await flush()

    heard.setProse('One.\nThe second thing said.\nThe third thing said.')
    await still()
    expect(written.length).toBe(1)

    heard.setProse('Two.\nThe second thing said.\nThe third thing said.')
    held.answer?.()
    await still()

    expect(written.length).toBe(2)
    expect(written[1]![0]!.text).toBe('Two.')
  })
})

describe('a transcript nobody edited', () => {
  it('is not written down when the editor hands back what it was given', async () => {
    const { recordings, written } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player, quiet: 5 })
    await flush()

    // The editor hands the document back carrying a newline of its own.
    heard.setProse(heard.prose.value + '\n')
    await still()

    expect(written).toStrictEqual([])
  })

  it('is written down once a word actually changes', async () => {
    const { recordings, written } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player, quiet: 5 })
    await flush()

    heard.setProse('The first thing Rupa said.\nThe second thing said.\nThe third thing said.')
    await still()

    expect(written).toHaveLength(1)
    expect(written[0]![0]!.text).toBe('The first thing Rupa said.')
    expect(written[0]![0]!.from).toBe(0)
  })
})

describe('the view going after the words', () => {
  it('stops while a person is typing, and starts again once they stop', async () => {
    const { recordings } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player, quiet: 5 })
    await flush()

    expect(heard.typing.value).toBe(false)

    heard.setProse('One.\nThe second thing said.\nThe third thing said.')
    expect(heard.typing.value).toBe(true)

    await still()
    expect(heard.typing.value).toBe(false)
  })

  it('is not stopped by the words arriving from the application', async () => {
    const { recordings } = talk()
    const heard = useTranscript(recordings, 'talks/Ants.mp3', { through: createPlayer().player, quiet: 5 })
    await flush()

    heard.setWorking(true)
    await flush()

    expect(heard.typing.value).toBe(false)
  })
})
