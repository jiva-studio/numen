/**
 * What the corner of the window draws, asked without a screen.
 *
 * The failure this is here for is one a person sees and no test did: one
 * document being read, and two cards about it carrying two counts that have
 * nothing to do with each other.
 */
import { describe, expect, it } from 'vitest'
import { getCornerNotices, type State, type Words } from './corner'
import type { Task } from './task'
import type { IndexCoverage } from './coverage'
import type { WindowMessage } from './messages'

const words: Words = {
  unwatched: 'not following the vault',
  unread: 'the vault could not be read',
  indexing: 'indexing the vault…',
  nothingRead: 'nothing was read',
  wordsOnly: 'Searching by words only — no model set',
}

const well = (over: Partial<State> = {}): State => ({
  unwatched: '',
  unread: '',
  lost: '',
  isIndexing: false,
  hasNote: true,
  ...over,
})

const vault = (over: Partial<IndexCoverage> = {}): IndexCoverage => ({
  chunks: 0,
  embedded: 0,
  isEmbedding: true,
  ...over,
})

const createTask = (over: Partial<Task> = {}): Task => ({
  id: 'indexing-1',
  label: 'Recognising a scan',
  about: 'library/Sabhaparva.pdf',
  done: 42,
  total: 400,
  counting: 'things',
  error: '',
  isUserStarted: true,
  ...over,
})

const createMessage = (over: Partial<WindowMessage> = {}): WindowMessage => ({
  id: 'command#1',
  name: 'command',
  kind: 'report',
  text: 'The note is in the trash',
  ...over,
})

const corner = (
  tasks: readonly Task[] = [],
  messages: readonly WindowMessage[] = [],
  state: State = well(),
  coverage: IndexCoverage = vault(),
) => getCornerNotices(tasks, messages, { state, vault: coverage, words })

describe('a document being read', () => {
  it('draws one card, whatever else the vault says about itself', () => {
    const rendered = corner([createTask()], [], well(), vault({ chunks: 65261 }))

    expect(rendered).toHaveLength(1)
    expect(rendered[0]?.text).toBe('Recognising a scan')
    expect(rendered[0]?.about).toBe('library/Sabhaparva.pdf')
    expect(rendered[0]).toMatchObject({ done: 42, total: 400 })
  })

  it('is drawn the moment it arrives, since a person requested for it', () => {
    expect(corner([createTask()])[0]?.isUserStarted).toBe(true)
  })

  it('waits to be drawn when nobody requested, since most such work is soon over', () => {
    const pass = createTask({
      id: 'extracting the books',
      label: 'Extracting books',
      isUserStarted: false,
    })

    expect(corner([pass])[0]?.isUserStarted).toBe(false)
  })
})

describe('work that stopped badly', () => {
  it('is drawn as alarm and not as something still running', () => {
    const rendered = corner([createTask({ error: 'nothing to read with' })])

    expect(rendered[0]?.tone).toBe('alarm')
    expect(rendered[0]?.isWorking).toBe(false)
  })

  it('is called by what stopped it, which is the sentence a person acts on', () => {
    const rendered = corner([createTask({ error: 'nothing to read with' })])

    expect(rendered[0]?.text).toBe('nothing to read with')
    expect(rendered[0]?.about).toBe('library/Sabhaparva.pdf')
  })

  it('is one card where one failure stopped several passes', () => {
    const error = 'intfloat/multilingual-e5-small is not on this machine'
    const rendered = corner([
      createTask({ id: 'getting ready', label: 'Preparing the model', error }),
      createTask({
        id: 'making the vectors',
        label: 'Indexing',
        error: `embedding demo: ${error}`,
      }),
    ])

    expect(rendered.map((one) => one.text)).toStrictEqual([error])
  })

  it('is one card where two passes stopped with the very same words', () => {
    const error = 'the model is not on this machine'
    const rendered = corner([
      createTask({ id: 'getting ready', label: 'Preparing the model', error }),
      createTask({ id: 'making the vectors', label: 'Indexing', error }),
    ])

    expect(rendered.map((one) => one.id)).toStrictEqual(['getting ready'])
  })

  it('keeps two failures that only happen to end in the same word', () => {
    const rendered = corner([
      createTask({ id: 'one', error: 'the disk is full' }),
      createTask({ id: 'two', error: 'the index says the disk is full' }),
    ])

    expect(rendered).toHaveLength(2)
  })

  it('is one card where the longer sentence arrived first', () => {
    const error = 'the model is not on this machine'
    const rendered = corner([
      createTask({
        id: 'making the vectors',
        label: 'Indexing',
        error: `embedding demo: ${error}`,
      }),
      createTask({ id: 'getting ready', label: 'Preparing the model', error }),
    ])

    expect(rendered.map((one) => one.id)).toStrictEqual(['getting ready'])
  })

  it('keeps a failure whose sentence ends where a pass running has no sentence yet', () => {
    const rendered = corner([
      createTask({ id: 'one', error: 'could not read: ' }),
      createTask({ id: 'two', error: '' }),
    ])

    expect(rendered.map((one) => one.id)).toStrictEqual(['one', 'two'])
  })

  it('keeps two failures that are two different things', () => {
    const rendered = corner([
      createTask({ id: 'one', error: 'nothing to read with' }),
      createTask({ id: 'two', error: 'the disk is full' }),
    ])

    expect(rendered).toHaveLength(2)
  })

  it('remains until it is put away, and is drawn at once even where nobody requested', () => {
    const rendered = corner([createTask({ isUserStarted: false, error: 'nothing to read with' })])

    expect(rendered[0]?.stay).toBe('kept')
    expect(rendered[0]?.isUserStarted).toBe(true)
  })
})

describe('work with nothing to count', () => {
  it('carries no tally, which is an ordinary state and not an unknown one', () => {
    const rendered = corner([createTask({ done: 0, total: 0 })])

    expect(rendered[0]?.done).toBeUndefined()
    expect(rendered[0]?.total).toBeUndefined()
    expect(rendered[0]?.isWorking).toBe(true)
  })

  it('draws no share for a model of a size nobody has been told', () => {
    const rendered = corner([
      createTask({
        id: 'getting ready',
        label: 'Fetching models',
        about: 'inference.onnx',
        done: 0,
        total: 0,
      }),
    ])

    expect(rendered[0]?.text).toBe('Fetching models')
    expect(rendered[0]?.total).toBeUndefined()
  })
})

describe('a step of a run', () => {
  const step = (what: string, about: string, count = 0, total = 0) =>
    createTask({
      id: 'making the vectors',
      label: what,
      about,
      done: count,
      total,
      isUserStarted: false,
    })

  it('is called by what it is, and names what it is on', () => {
    const rendered = corner([step('Indexing', 'library/Sabhaparva.epub', 300, 1200)])

    expect(rendered[0]?.text).toBe('Indexing')
    expect(rendered[0]?.about).toBe('library/Sabhaparva.epub')
    expect(rendered[0]).toMatchObject({ done: 300, total: 1200 })
  })

  it('names what it moved on to, and how much further it got', () => {
    const one = corner([step('Indexing', 'library/Sabhaparva.epub', 300, 1200)])
    const next = corner([step('Indexing', 'notes/Vrindavan.md', 900, 1200)])

    expect(one[0]?.about).toBe('library/Sabhaparva.epub')
    expect(next[0]?.about).toBe('notes/Vrindavan.md')
    expect(next[0]?.done).toBe(900)
  })
})

describe('what is so about the window', () => {
  it('says the vault is not being followed, and which vault it is', () => {
    const rendered = corner([], [], well({ unwatched: '/home/vault' }))

    expect(rendered).toHaveLength(1)
    expect(rendered[0]?.text).toBe(words.unwatched)
    expect(rendered[0]?.about).toBe('/home/vault')
    expect(rendered[0]?.stay).toBe('holds')
    expect(rendered[0]?.tone).toBe('caution')
  })

  it('says nothing was read only once the indexing is over', () => {
    const rendered = corner(
      [],
      [],
      well({ isIndexing: true, hasNote: false, unread: 'no such folder' }),
    )

    expect(rendered.map((one) => one.text)).toStrictEqual([words.unread, words.indexing])
  })

  it('says the vault could not be read, and that nothing was read from it', () => {
    const rendered = corner([], [], well({ unread: 'the vault folder is not there', hasNote: false }))

    expect(rendered.map((one) => one.text)).toStrictEqual([words.unread, words.nothingRead])
  })

  it('says nothing was read only where the vault could not be', () => {
    const rendered = corner([], [], well({ hasNote: false }))

    expect(rendered).toStrictEqual([])
  })

  it('says the vault is still being read for the first time', () => {
    const rendered = corner([], [], well({ isIndexing: true }))

    expect(rendered.map((one) => one.text)).toStrictEqual([words.indexing])
  })

  it('says what the window lost touch with, in the words it was given', () => {
    const rendered = corner([], [], well({ lost: 'the themes stopped arriving' }))

    expect(rendered.map((one) => one.text)).toStrictEqual(['the themes stopped arriving'])
  })

  it('is drawn at once, since nothing about it is going to last ten seconds first', () => {
    expect(corner([], [], well({ unwatched: '/home/vault' }))[0]?.isUserStarted).toBe(true)
  })
})

describe('each state of the window', () => {
  it('is drawn under its own identity, tone and manner', () => {
    const rendered = corner(
      [],
      [],
      well({ unwatched: '/home/vault', unread: 'no such folder', lost: 'the themes stopped' }),
    )

    expect(rendered).toStrictEqual([
      {
        id: 'unwatched',
        text: words.unwatched,
        about: '/home/vault',
        tone: 'caution',
        isWorking: false,
        isUserStarted: true,
        stay: 'holds',
      },
      {
        id: 'unread',
        text: words.unread,
        about: 'no such folder',
        tone: 'caution',
        isWorking: false,
        isUserStarted: true,
        stay: 'holds',
      },
      {
        id: 'lost',
        text: 'the themes stopped',
        about: '',
        tone: 'caution',
        isWorking: false,
        isUserStarted: true,
        stay: 'holds',
      },
    ])
  })

  it('draws the first indexing plainly under its own identity', () => {
    expect(corner([], [], well({ isIndexing: true }))).toStrictEqual([
      {
        id: 'indexing',
        text: words.indexing,
        about: '',
        tone: 'plain',
        isWorking: false,
        isUserStarted: true,
        stay: 'holds',
      },
    ])
  })

  it('draws nothing read plainly under its own identity', () => {
    const rendered = corner([], [], well({ unread: 'gone', hasNote: false }))

    expect(rendered[1]).toStrictEqual({
      id: 'nothingRead',
      text: words.nothingRead,
      about: '',
      tone: 'plain',
      isWorking: false,
      isUserStarted: true,
      stay: 'holds',
    })
  })

  it('draws the words-only state quietly under its own identity', () => {
    const rendered = corner([], [], well(), vault({ chunks: 4823, isEmbedding: false }))

    expect(rendered).toStrictEqual([
      {
        id: 'wordsOnly',
        text: words.wordsOnly,
        about: '',
        tone: 'plain',
        isWorking: false,
        isUserStarted: false,
        stay: 'holds',
      },
    ])
  })
})

describe('what the window returned', () => {
  it('is never work in progress', () => {
    expect(corner([], [createMessage()])[0]?.isWorking).toBe(false)
  })

  it('draws a report to be read and then release', () => {
    const rendered = corner([], [createMessage()])

    expect(rendered[0]).toMatchObject({
      text: 'The note is in the trash',
      tone: 'plain',
      stay: 'read',
    })
  })

  it('draws a caution to be read and left standing', () => {
    const rendered = corner(
      [],
      [createMessage({ kind: 'caution', text: 'that note changed on disk' })],
    )

    expect(rendered[0]).toMatchObject({ tone: 'caution', stay: 'kept' })
  })

  it('draws an error that remains until it is put away', () => {
    const rendered = corner(
      [],
      [createMessage({ kind: 'error', text: 'a note of that name is filed there' })],
    )

    expect(rendered[0]).toMatchObject({ tone: 'alarm', stay: 'kept' })
  })

  it('draws a state for as long as whoever returned it keeps saying it', () => {
    const rendered = corner(
      [],
      [createMessage({ kind: 'state', text: 'the themes stopped arriving' })],
    )

    expect(rendered[0]).toMatchObject({ tone: 'plain', stay: 'holds' })
  })

  it('keeps each utterance under the identity it was given', () => {
    const rendered = corner([], [createMessage({ id: 'command#7' })])

    expect(rendered[0]?.id).toBe('command#7')
  })
})

describe('the order the corner draws in', () => {
  it('puts work first, so a state or a word arriving does not shift a moving count', () => {
    const rendered = corner([createTask()], [createMessage()], well({ unwatched: '/home/vault' }))

    expect(rendered.map((one) => one.id)).toStrictEqual(['indexing-1', 'unwatched', 'command#1'])
  })
})

describe('chunks with nothing to embed them', () => {
  it('says the search is by words alone, since nothing is going to bring the rest', () => {
    const rendered = corner([], [], well(), vault({ chunks: 4823, isEmbedding: false }))

    expect(rendered).toHaveLength(1)
    expect(rendered[0]?.text).toBe(words.wordsOnly)
    expect(rendered[0]?.isWorking).toBe(false)
  })

  it('says nothing where a model is going to embed them', () => {
    expect(corner([], [], well(), vault({ chunks: 4823, isEmbedding: true }))).toEqual([])
  })

  it('says nothing about a vault that holds nothing', () => {
    expect(corner([], [], well(), vault({ isEmbedding: false }))).toEqual([])
  })
})
