/**
 * Carrying a command out, asked without a window.
 *
 * Two things are asked here: that a note is settled before its file is renamed
 * or removed, and that what a remove leaves behind is put to the person.
 */
import { describe, expect, it } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { commandsOf } from '../lib/commands'
import { runSupport } from './runs'
import { invocationOf } from '../lib/invocation'
import { createNotes } from '../lib/notes'
import type { VaultRef } from '../types'
import type { CommandDeps } from './deps'
import type { CommandInvocation, CommandTarget, Store } from '../types'
import { runInvocation } from './handlers'
import type { Artifact, ArtifactStates, Outcome, ArtifactState } from '@/entities/artifact'
import { asFailure, asValue } from '@numen/wire'
import type { Movement } from '@/entities/file'
import type { RemoveResult, RenameResult } from '@/entities/note'
import type { Vault, VaultErrorCode, VaultResult } from '@/entities/vault'
import type { ErrorCode } from '@/shared/errors'
import { WORDS } from '@/shared/words'
import { ANSWER_WORDS as words } from '../words'

/** What is in front, which every invocation is carried out over. */
const front = (over: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: 'tab',
  kind: 'plex',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: '',
  source: null,
  made: {},
  vault: { id: 'physics', name: 'Physics' },
  ready: true,
  ...over,
})

/** One vault as the list answers one. */
const createVault = (id: string, name: string): Vault => ({
  id,
  name,
  path: `/vaults/${name}`,
  missing: false,
})

const createRenameResult = (over: Partial<RenameResult> = {}): RenameResult => ({
  path: 'physics/Entropy.md',
  title: 'Entropy',
  hasFrontmatter: false,
  moved: null,
  error: null,
  hasChanged: false,
  ...over,
})

/** What an artifact now stands at, as the application answers it. */
const outcome = (of: Artifact, state: ArtifactState, error = ''): Outcome => ({
  able: true,
  of,
  made: state,
  error,
})

const createRemoveResult = (over: Partial<RemoveResult> = {}): RemoveResult => ({
  trashed: '.trash/Ontology.md',
  dangling: [],
  error: null,
  ...over,
})

/**
 * A window that writes down everything a command asked of it, in order.
 *
 * The note the window holds stands at a file the test can move, so an invocation made
 * before it moved can be carried out after.
 */
const window = (
  answers: {
    renamed?: RenameResult
    removed?: RemoveResult
    made?: boolean
    /** Where the tab holding the note stands now. */
    at?: string
    /** The note is waiting on the person, so nothing may move its file. */
    asking?: boolean
    /** The folder the person chose in the machine's own picker. */
    chose?: string
    /** What the list of vaults answered adding or renaming one. */
    added?: VaultResult
    /** What the list of vaults refused forgetting, erasing or opening one. */
    turnedDown?: VaultErrorCode
    /** What moving a file came back with. */
    movement?: Movement
    /** What making a folder came back with. */
    folderError?: ErrorCode
    /** What the file in front carries. */
    carries?: ArtifactStates
    /** How asking for an artifact of a file came out. */
    outcome?: Outcome
    /** What deleting the text came back with. */
    deleteError?: string
    /** Whether this build cannot delete the text at all. */
    undeletable?: boolean
  } = {},
) => {
  const done: string[] = []
  const said: string[] = []
  /** The voice each sentence was said in, one to a sentence. */
  const tones: string[] = []
  const at = answers.at ?? 'physics/Ontology.md'
  const turnedDown = answers.turnedDown ?? null
  /** The runs this one window has been told this build cannot do. */
  const runs = runSupport()
  const on: CommandDeps = {
    files: {
      createNote: async (title, from, seat) => {
        done.push(`createNote ${title} ${from || '—'} ${seat ?? '—'}`)
        return answers.made === false ? null : { path: `${title}.md`, title }
      },
      rename: async (path, title) => {
        done.push(`rename ${path} ${title}`)
        return answers.renamed ?? createRenameResult()
      },
      remove: async (path, destroy) => {
        done.push(`remove ${path} ${destroy}`)
        return answers.removed ?? createRemoveResult()
      },
      move: async (from, to) => {
        done.push(`move ${from} ${to}`)
        return answers.movement ?? { ok: true, value: null }
      },
      createFolder: async (path) => {
        done.push(`createFolder ${path}`)
        return answers.folderError ?? null
      },
    },
    runs: {
      getArtifactStates: async (path) => {
        done.push(`carries ${path}`)
        return answers.carries ?? {}
      },
      correctArtifact: async (path) => {
        done.push(`corrects ${path}`)
        return answers.outcome ?? outcome('transcript.corrected', 'running')
      },
      fetchArtifact: async (path) => {
        done.push(`fetches ${path}`)
        return answers.outcome ?? outcome('transcript', 'running')
      },
      createArtifact: async (path, of) => {
        done.push(`makes ${of} ${path}`)
        return answers.outcome ?? outcome(of, 'running')
      },
      deleteTranscript: async (path: string) => {
        done.push(`deletes the text of ${path}`)
        if (answers.deleteError)
          throw new ConnectError(answers.deleteError, Code.FailedPrecondition)
        return answers.undeletable !== true
      },
      deleteCopy: async (path: string) => {
        done.push(`deletes the copy of ${path}`)
        return answers.undeletable !== true
      },
    },
    makers: {
      createDeck: async (folder, name) => {
        done.push(`decks ${folder || '—'} ${name}`)
        return folder ? `${folder}/${name}` : name
      },
      createStencil: async (folder, name) => {
        done.push(`stencils ${folder || '—'} ${name}`)
        return folder ? `${folder}/${name}` : name
      },
      createPreset: async (folder, name) => {
        done.push(`presets ${folder || '—'} ${name}`)
        return folder ? `${folder}/${name}` : name
      },
      createUrl: async (folder, address) => {
        done.push(`imports ${folder || '—'} ${address}`)
        return folder ? `${folder}/note.md` : 'note.md'
      },
    },
    notes: {
      getTabAt: (path) => (path === at ? 'held' : null),
      getPath: (id) => (id === 'held' ? at : id),
      isAsking: () => answers.asking === true,
      settle: async (id) => void done.push(`settle ${id}`),
      close: (id) => void done.push(`close ${id}`),
      openFile: (path, title, showing) => void done.push(`openFile ${path} ${title} ${showing}`),
      openNewFile: (path, title, type, showing) =>
        void done.push(`openNewFile ${type} ${path} ${title} ${showing}`),
    },
    vaults: {
      list: async () => ({ vaults: [createVault('physics', 'Physics')], showing: 'physics' }),
      choose: async (title) => {
        done.push(`choose ${title}`)
        return answers.chose ?? '/vaults/Heat'
      },
      add: async (path, name) => {
        done.push(`add ${path} ${name || '—'}`)
        return answers.added ?? asValue(createVault('heat', 'Heat'))
      },
      rename: async (id, name) => {
        done.push(`renames vault ${id} ${name}`)
        return answers.added ?? asValue(createVault(id, name))
      },
      remove: async (id, trash) => {
        done.push(trash ? `erases ${id}` : `forgets ${id}`)
        return turnedDown
      },
      open: async (id) => {
        done.push(`opens vault ${id}`)
        return turnedDown
      },
      setVaultName: (vault: VaultRef) => void done.push(`calls ${vault.id} ${vault.name}`),
      reload: () => void done.push('reloads'),
    },
    goes: {
      revealPath: (path) => void done.push(`revealPath ${path}`),
      travel: async (path) => void done.push(`travel ${path}`),
      leave: async (from, to) => void done.push(`leave ${from} ${to}`),
      getOpeningNote: () => 'Root.md',
      openTab: (kind) => void done.push(`openTab ${kind}`),
      openPreset: async (path) => void done.push(`preset ${path}`),
      closeTab: (tab) => void done.push(`closeTab ${tab}`),
      ask: (text) => void done.push(`ask ${text}`),
      search: () => void done.push('search'),
    },
    settings: {
      chooseAppearance: async (chosen) => void done.push(`appearance ${chosen}`),
      chooseSync: async (chosen) => void done.push(`syncing ${chosen}`),
      chooseHanging: async (chosen) => void done.push(`hanging ${chosen}`),
      chooseParts: async (chosen) => void done.push(`parts ${chosen}`),
    },
    runSupport: runs,
    copyPath: (path) => void done.push(`copies ${path}`),
    writeMessage: (text, kind) => {
      if (!text) return
      said.push(text)
      tones.push(kind ?? '')
    },
  }
  return { on, done, said, tones, runs }
}

/** One command carried out over the note in front. */
const carry = async (invocation: CommandInvocation, on: CommandDeps) =>
  runInvocation(invocation, on, words)

describe('every command that is offered', () => {
  it('is carried out by something', async () => {
    for (const command of commandsOf(WORDS)) {
      const one = window()
      await carry(invocationOf(command.id, front(), 'Entropy'), one.on)

      expect(one.done, command.id).not.toStrictEqual([])
    }
  })
})

describe('a note put in front of the person', () => {
  it('opens in a tab of its own, and beside it where the second key reached it', async () => {
    const one = window()

    await carry(invocationOf('read', front()), one.on)
    await carry(invocationOf('beside', front()), one.on)

    expect(one.done).toStrictEqual([
      'openFile physics/Ontology.md Ontology here',
      'openFile physics/Ontology.md Ontology beside',
    ])
  })

  it('is travelled to in the plex, where that is what was asked', async () => {
    const one = window()

    await carry(invocationOf('travel', front()), one.on)

    expect(one.done).toStrictEqual(['travel physics/Ontology.md'])
  })
})

describe('a deck, a stencil or a preset made', () => {
  it('is made at the top of the vault, under the name as it was typed', async () => {
    const one = window()

    await carry(invocationOf('deck', front(), 'Animals'), one.on)

    // The vault names the file, so nothing here puts an ending on the name.
    expect(one.done).toStrictEqual(['decks — Animals'])
  })

  it('hands a name that carries an ending over unchanged', async () => {
    const one = window()

    await carry(invocationOf('deck', front(), 'Animals.md'), one.on)

    expect(one.done).toStrictEqual(['decks — Animals.md'])
  })

  it('is a stencil where that is what was asked for', async () => {
    const one = window()

    await carry(invocationOf('stencil', front(), 'Animal'), one.on)

    expect(one.done).toStrictEqual(['stencils — Animal'])
  })

  it('is a preset where that is what was asked for', async () => {
    const one = window()

    await carry(invocationOf('newPreset', front(), 'Sanskrit'), one.on)

    expect(one.done).toStrictEqual(['presets — Sanskrit'])
  })

  it('is nothing at all where nothing was typed', async () => {
    const one = window()

    await carry(invocationOf('deck', front(), ''), one.on)
    await carry(invocationOf('newPreset', front(), ''), one.on)

    expect(one.done).toStrictEqual([])
  })
})

describe('a note made', () => {
  it('writes the note it was made from into it, in the seat that was asked for', async () => {
    const one = window()

    await carry(invocationOf('child', front(), 'Entropy'), one.on)

    expect(one.done[0]).toBe('createNote Entropy physics/Ontology.md child')
  })

  it('stands on its own where no seat was asked for', async () => {
    const one = window()

    await carry(invocationOf('note', front(), 'Entropy'), one.on)

    expect(one.done[0]).toBe('createNote Entropy — —')
  })

  it('is travelled to in the plex the person is looking at', async () => {
    const one = window()

    await carry(invocationOf('child', front(), 'Entropy'), one.on)

    expect(one.done.at(-1)).toBe('travel Entropy.md')
  })

  it('opens in a tab beside the note the person is in', async () => {
    const one = window()

    await carry(invocationOf('child', front({ kind: 'note' }), 'Entropy'), one.on)

    expect(one.done.at(-1)).toBe('openNewFile note Entropy.md Entropy beside')
  })

  it('takes the person nowhere where the vault would not make it', async () => {
    const one = window({ made: false })

    await carry(invocationOf('child', front(), 'Entropy'), one.on)

    expect(one.done).toStrictEqual(['createNote Entropy physics/Ontology.md child'])
  })

  it('is nothing at all where nothing was typed', async () => {
    const one = window()

    await carry(invocationOf('child', front(), ''), one.on)

    expect(one.done).toStrictEqual([])
  })
})

/**
 * What the window says an artifact of a file now stands at, as a report. Work
 * that came off is one of these: only what did not is an error.
 */
const REPORTED: readonly ArtifactState[] = ['queued', 'running', 'done']

/** Everything an artifact can stand at, which the window has a sentence for. */
const REACHED: readonly ArtifactState[] = [
  'none',
  'queued',
  'running',
  'stopped',
  'done',
  'empty',
  'failed',
]

/** An artifact of the recording in front asked for, as it came out. */
const createArtifactRun = (state: ArtifactState, error = '') => {
  const one = window({ outcome: outcome('transcript', state, error) })
  return { one, invocation: invocationOf('transcribe', front({ file: 'talks/Ants.mp3' })) }
}

describe('an artifact asked for over a file', () => {
  it('asks the application over that file', async () => {
    const { one, invocation } = createArtifactRun('running')

    await carry(invocation, one.on)

    expect(one.done).toStrictEqual(['makes transcript talks/Ants.mp3'])
  })

  // Every state carries a sentence, so the person is told one whatever
  // happened.
  it('says what it now stands at, in the window’s own words', async () => {
    for (const made of REACHED) {
      const { one, invocation } = createArtifactRun(made)

      await carry(invocation, one.on)

      expect(one.said, made).toStrictEqual([words.made.transcript[made]])
    }
  })

  // Only a run under way is a report. Everything else is an error: the person
  // asked for work, and none is being done.
  it('says a run under way as a report and the rest as errors', async () => {
    for (const made of REACHED) {
      const { one, invocation } = createArtifactRun(made)

      await carry(invocation, one.on)

      expect(one.tones, made).toStrictEqual([REPORTED.includes(made) ? 'report' : 'error'])
    }
  })

  // The person asked for this one by name, and a recording already transcribed
  // would otherwise look like a command that did nothing.
  it('says a recording already transcribed has been transcribed', async () => {
    const { one, invocation } = createArtifactRun('done')

    await carry(invocation, one.on)

    expect(one.said).toStrictEqual([words.made.transcript.done])
  })

  // A run that could not read the file wrote down what it got, and asking again
  // gets the same until that record is taken away.
  it('says what a run said about a recording it could not open', async () => {
    const said = 'mp3: MPEG version 2.5 is not supported'
    const { one, invocation } = createArtifactRun('failed', said)

    await carry(invocation, one.on)

    expect(one.done).toStrictEqual(['makes transcript talks/Ants.mp3'])
    expect(one.said).toStrictEqual([`${words.made.transcript.failed} ${said}`])
  })

  it('says the same of a scan already recognised', async () => {
    const one = window({ outcome: outcome('ocr', 'done') })

    await carry(invocationOf('recognise', front({ file: 'books/Ants.pdf' })), one.on)

    expect(one.done).toStrictEqual(['makes ocr books/Ants.pdf'])
    expect(one.said).toStrictEqual([words.made.ocr.done])
  })

  // No two of them may say the same thing: a person reads the sentence and not
  // the word behind it.
  it('says something of its own for every artifact and every state', () => {
    const said = Object.values(words.made).flatMap((one) => Object.values(one))

    expect(said.filter((one) => one === '')).toStrictEqual([])
    expect(new Set(said).size).toBe(said.length)
  })

  it('says this build cannot do it, and offers it nowhere after that', async () => {
    const one = window({ outcome: { able: false } })

    await carry(invocationOf('transcribe', front({ file: 'talks/Ants.mp3' })), one.on)

    expect(one.said).toStrictEqual([words.unrunnable])
    expect(one.runs.canRun('transcribe')).toBe(false)
    expect(one.runs.canRun('recognise')).toBe(true)
  })

  it('is said only in the window it was asked in', async () => {
    const one = window({ outcome: { able: false } })

    await carry(invocationOf('transcribe', front({ file: 'talks/Ants.mp3' })), one.on)

    expect(window().runs.canRun('transcribe')).toBe(true)
  })
})

describe('the transcript of a recording deleted', () => {
  it('asks the application over the file the tab in front holds', async () => {
    const one = window()

    await carry(invocationOf('deleteText', front({ file: 'talks/Ants.mp3' })), one.on)

    expect(one.done).toStrictEqual(['deletes the text of talks/Ants.mp3'])
    expect(one.said).toStrictEqual([])
  })

  it('says what the application refused, in the words it sent', async () => {
    const why = 'this recording is being listened to'
    const one = window({ deleteError: why })

    await carry(invocationOf('deleteText', front({ file: 'talks/Ants.mp3' })), one.on)

    expect(one.said).toStrictEqual([why])
    expect(one.tones).toStrictEqual(['error'])
  })

  it('says this build cannot do it, and offers it nowhere after that', async () => {
    const one = window({ undeletable: true })

    await carry(invocationOf('deleteText', front({ file: 'talks/Ants.mp3' })), one.on)

    expect(one.said).toStrictEqual([words.unrunnable])
    expect(one.runs.canRun('deleteText')).toBe(false)
  })
})

describe('a note renamed', () => {
  it('has nothing on its way to its file before the file moves', async () => {
    const one = window()

    await carry(invocationOf('title', front(), 'Entropy'), one.on)

    expect(one.done).toStrictEqual(['settle held', 'rename physics/Ontology.md Entropy'])
  })

  it('is refused while the note is waiting on the person', async () => {
    const one = window({ asking: true })

    await carry(invocationOf('title', front(), 'Entropy'), one.on)

    expect(one.done).toStrictEqual([])
    expect(one.said).toStrictEqual([words.unanswered])
  })

  it('says the note was written elsewhere while this was asked', async () => {
    const one = window({ renamed: createRenameResult({ hasChanged: true }) })

    await carry(invocationOf('title', front(), 'Entropy'), one.on)

    expect(one.said).toStrictEqual([words.stale])
  })

  it('is renamed where no tab of the window holds it', async () => {
    const one = window()

    await carry(invocationOf('title', front({ path: 'Elsewhere.md' }), 'Entropy'), one.on)

    expect(one.done).toStrictEqual(['rename Elsewhere.md Entropy'])
  })

  it('is left alone where the name it was given is the name it has', async () => {
    const one = window()

    await carry(invocationOf('title', front(), 'Ontology'), one.on)

    expect(one.done).toStrictEqual([])
  })

  it('says nothing of the notes whose links it wrote again', async () => {
    const one = window({
      renamed: createRenameResult({
        moved: {
          from: 'physics/Ontology.md',
          to: 'physics/Entropy.md',
          repaired: ['Notes.md', 'Order.md'],
        },
      }),
    })

    await carry(invocationOf('title', front(), 'Entropy'), one.on)

    expect(one.said).toStrictEqual([])
  })

  it('says nothing of the title it wrote into the frontmatter', async () => {
    const one = window({ renamed: createRenameResult({ hasFrontmatter: true }) })

    await carry(invocationOf('title', front(), 'Entropy'), one.on)

    expect(one.said).toStrictEqual([])
  })

  it('says the name was taken, and that the note carries the new one', async () => {
    const one = window({ renamed: createRenameResult({ error: 'occupied' }) })

    await carry(invocationOf('title', front(), 'Entropy'), one.on)

    expect(one.said).toStrictEqual([words.errors.occupied])
  })

  it('says a note whose frontmatter cannot be read cannot be renamed', async () => {
    const one = window({ renamed: createRenameResult({ error: 'unreadable' }) })

    await carry(invocationOf('title', front(), 'Entropy'), one.on)

    expect(one.said).toStrictEqual([words.errors.unreadable])
  })

  it('says a name no file can be named', async () => {
    const one = window({ renamed: createRenameResult({ error: 'unnameable' }) })

    await carry(invocationOf('title', front(), '...'), one.on)

    expect(one.said).toStrictEqual([words.errors.unnameable])
  })
})

describe('a note removed', () => {
  it('has nothing on its way to its file before the file goes', async () => {
    const one = window()

    await carry(invocationOf('remove', front()), one.on)

    expect(one.done.slice(0, 2)).toStrictEqual(['settle held', 'remove physics/Ontology.md false'])
  })

  it('goes off the disk where destroying was what was asked', async () => {
    const one = window()

    await carry(invocationOf('destroy', front(), 'Ontology'), one.on)

    expect(one.done[1]).toBe('remove physics/Ontology.md true')
  })

  /** The row has gone from the tree, which is the whole of what a person needs. */
  it('says nothing about a note that went where it was asked to go', async () => {
    const one = window()

    await carry(invocationOf('remove', front()), one.on)

    expect(one.said).toStrictEqual([])
  })

  it('says the notes that link to nothing now', async () => {
    const one = window({ removed: createRemoveResult({ dangling: ['Order.md', 'Notes.md'] }) })

    await carry(invocationOf('remove', front()), one.on)

    expect(one.said).toStrictEqual([`${words.dangling} Order.md, Notes.md`])
  })

  it('leaves every plex standing on it at the note the vault opens with', async () => {
    const one = window()

    await carry(invocationOf('remove', front()), one.on)

    expect(one.done.at(-1)).toBe('leave physics/Ontology.md Root.md')
  })

  it('leaves the plexes alone where the vault opens with no note at all', async () => {
    const one = window()
    const nowhere: CommandDeps = { ...one.on, goes: { ...one.on.goes, getOpeningNote: () => '' } }

    await carry(invocationOf('remove', front()), nowhere)

    expect(one.done.some((step) => step.startsWith('leave'))).toBe(false)
  })

  it('lets go of the tab that was reading it', async () => {
    const one = window()

    await carry(invocationOf('remove', front(), '', 'held'), one.on)

    expect(one.done).toContain('close held')
  })

  it('keeps the tab of a note the vault would not remove', async () => {
    const one = window({ removed: createRemoveResult({ error: 'missing' }) })

    await carry(invocationOf('remove', front(), '', 'held'), one.on)

    expect(one.done).not.toContain('close held')
  })

  it('says a note that is not in the vault, and takes the plex nowhere', async () => {
    const one = window({ removed: createRemoveResult({ error: 'missing' }) })

    await carry(invocationOf('remove', front()), one.on)

    expect(one.said).toStrictEqual([words.errors.missing])
    expect(one.done.at(-1)).toBe('remove physics/Ontology.md false')
  })

  it('is refused while the note is waiting on the person', async () => {
    const one = window({ asking: true })

    await carry(invocationOf('remove', front()), one.on)

    expect(one.done).toStrictEqual([])
    expect(one.said).toStrictEqual([words.unanswered])
  })
})

describe('several files removed at once', () => {
  const both = front({ others: ['physics/Heat.pdf'] })

  it('takes each of them out of the vault, in the order they were given', async () => {
    const one = window()

    await carry(invocationOf('remove', both), one.on)

    expect(one.done.filter((step) => step.startsWith('remove'))).toStrictEqual([
      'remove physics/Ontology.md false',
      'remove physics/Heat.pdf false',
    ])
  })

  it('takes every plex standing on one of them to the note the vault opens with', async () => {
    const one = window()

    await carry(invocationOf('remove', both), one.on)

    expect(one.done.filter((step) => step.startsWith('leave'))).toStrictEqual([
      'leave physics/Ontology.md Root.md',
      'leave physics/Heat.pdf Root.md',
    ])
  })

  it('says the notes that link to nothing now, each of them once', async () => {
    const one = window({ removed: createRemoveResult({ trashed: '', dangling: ['Order.md'] }) })

    await carry(invocationOf('remove', both), one.on)

    expect(one.said).toStrictEqual([`${words.dangling} Order.md`])
  })

  it('takes the rest out where the vault refuses one, and says what it refused', async () => {
    const one = window()
    const picky: CommandDeps = {
      ...one.on,
      files: {
        ...one.on.files,
        remove: async (path, destroy) => {
          one.done.push(`remove ${path} ${destroy}`)
          return createRemoveResult(path === 'physics/Ontology.md' ? { error: 'missing' } : {})
        },
      },
    }

    await carry(invocationOf('remove', both), picky)

    expect(one.done.filter((step) => step.startsWith('remove'))).toStrictEqual([
      'remove physics/Ontology.md false',
      'remove physics/Heat.pdf false',
    ])
    expect(one.said).toStrictEqual([words.errors.missing])
  })
})

describe('a file filed somewhere else', () => {
  /** The destination is the whole path, so a name changed in one folder is a move. */
  const createMoveInvocation = (to: string) => invocationOf('move', front(), to)

  it('is asked of the vault under the path it is filed at from now on', async () => {
    const one = window()

    await carry(createMoveInvocation('notes/Ontology.md'), one.on)

    expect(one.done).toStrictEqual(['settle held', 'move physics/Ontology.md notes/Ontology.md'])
  })

  it('settles the tab holding it before its file goes anywhere', async () => {
    const one = window()

    await carry(createMoveInvocation('notes/Ontology.md'), one.on)

    expect(one.done.indexOf('settle held')).toBeLessThan(
      one.done.indexOf('move physics/Ontology.md notes/Ontology.md'),
    )
  })

  it('stays where it is while its tab is waiting on the person', async () => {
    const one = window({ asking: true })

    await carry(createMoveInvocation('notes/Ontology.md'), one.on)

    expect(one.done).toStrictEqual([])
    expect(one.said).toStrictEqual([words.unanswered])
  })

  it('stays where it is where something of that name is filed there', async () => {
    const one = window({ movement: { ok: false, error: 'occupied' } })

    await carry(createMoveInvocation('notes/Ontology.md'), one.on)

    expect(one.said).toStrictEqual([words.occupied])
  })

  it('says nothing of a note renamed, which is what a move is not', async () => {
    const one = window({ movement: { ok: false, error: 'occupied' } })

    await carry(createMoveInvocation('notes/Ontology.md'), one.on)

    expect(one.said).not.toContain(words.errors.occupied)
  })

  it('asks the vault for nothing where it landed where it already was', async () => {
    const one = window()

    await carry(createMoveInvocation('physics/Ontology.md'), one.on)

    expect(one.done).toStrictEqual([])
  })
})

describe('a folder made', () => {
  it('is asked of the vault under the path it goes at', async () => {
    const one = window()

    await carry(invocationOf('createFolder', front(), 'physics/heat'), one.on)

    expect(one.done).toStrictEqual(['createFolder physics/heat'])
  })

  it('is not made where something of that name is filed there', async () => {
    const one = window({ folderError: 'occupied' })

    await carry(invocationOf('createFolder', front(), 'physics/heat'), one.on)

    expect(one.said).toStrictEqual([words.occupied])
  })

  it('asks the vault for nothing where no path was given', async () => {
    const one = window()

    await carry(invocationOf('createFolder', front()), one.on)

    expect(one.done).toStrictEqual([])
  })
})

/**
 * An invocation is made when a person answers and carried out a moment later, and the
 * vault moves in between. The tab holding the note is what says where it is.
 */
describe('a note that moved between the answer and the invocation', () => {
  it('is renamed where it stands now, not at the name the invocation was made over', async () => {
    const one = window({ at: 'physics/Being.md' })

    await carry(invocationOf('title', front(), 'Substance', 'held'), one.on)

    expect(one.done).toStrictEqual(['settle held', 'rename physics/Being.md Substance'])
  })

  it('is removed where it stands now', async () => {
    const one = window({ at: 'physics/Being.md' })

    await carry(invocationOf('remove', front(), '', 'held'), one.on)

    expect(one.done.slice(0, 2)).toStrictEqual(['settle held', 'remove physics/Being.md false'])
  })

  it('is left at the name it was made over where no tab holds it', async () => {
    const one = window({ at: 'physics/Being.md' })

    await carry(invocationOf('remove', front()), one.on)

    expect(one.done[0]).toBe('remove physics/Ontology.md false')
  })
})

describe('a command over the window', () => {
  it('opens a tab of the kind asked for, and closes the one in front', async () => {
    const one = window()

    await carry(invocationOf('plex', front()), one.on)
    await carry(invocationOf('agent', front()), one.on)
    await carry(invocationOf('close', front()), one.on)

    expect(one.done).toStrictEqual(['openTab plex', 'openTab agent', 'closeTab tab'])
  })

  it('hands the field back to the search', async () => {
    const one = window()

    await carry(invocationOf('find', front()), one.on)

    expect(one.done).toStrictEqual(['search'])
  })

  it('hands over the row that was chosen, and nothing about the note in front', async () => {
    const one = window()

    await carry(invocationOf('appearance', front(), 'mine:sea'), one.on)
    await carry(invocationOf('appearance', front(), 'mode:dark'), one.on)

    expect(one.done).toStrictEqual(['appearance mine:sea', 'appearance mode:dark'])
  })
})

describe('a command over the vault', () => {
  it('travels to the note the vault opens with', async () => {
    const one = window()

    await carry(invocationOf('first', front()), one.on)

    expect(one.done).toStrictEqual(['travel Root.md'])
  })

  it('says a vault that opens with no note at all', async () => {
    const one = window()
    const empty: CommandDeps = { ...one.on, goes: { ...one.on.goes, getOpeningNote: () => '' } }

    await runInvocation(invocationOf('first', front()), empty, words)

    expect(one.said).toStrictEqual([words.nowhere])
  })
})

describe('what the window is asked about a note', () => {
  it('is put to the agent, and its path put on the clipboard', async () => {
    const one = window()

    await carry(invocationOf('ask', front()), one.on)
    await carry(invocationOf('copy', front()), one.on)

    expect(one.done).toStrictEqual(['ask physics/Ontology.md — ', 'copies physics/Ontology.md'])
  })
})

describe('another vault under this window', () => {
  const heat = () => front({ vault: { id: 'heat', name: 'Heat' } })

  it('is opened, and the page drawn again on it', async () => {
    const one = window()

    await carry(invocationOf('openVault', heat()), one.on)

    expect(one.done).toStrictEqual(['opens vault heat', 'reloads'])
  })

  it('leaves the page where it stands where the vault would not open', async () => {
    const one = window({ turnedDown: 'showing' })

    await carry(invocationOf('openVault', heat()), one.on)

    expect(one.done).toStrictEqual(['opens vault heat'])
    expect(one.said).toStrictEqual([words.vaultErrors.showing])
  })
})

describe('a vault made', () => {
  it('is the folder chosen in the machine’s own picker, and is opened', async () => {
    const one = window()

    await carry(invocationOf('newVault', front()), one.on)

    expect(one.done).toStrictEqual([
      `choose ${words.folder}`,
      'add /vaults/Heat —',
      'opens vault heat',
      'reloads',
    ])
  })

  it('is nothing at all where the person closed the picker', async () => {
    const one = window({ chose: '' })

    await carry(invocationOf('newVault', front()), one.on)

    expect(one.done).toStrictEqual([`choose ${words.folder}`])
    expect(one.said).toStrictEqual([])
  })

  it('says a folder that lies inside a vault already added', async () => {
    const one = window({ added: asFailure('overlaps') })

    await carry(invocationOf('newVault', front()), one.on)

    expect(one.said).toStrictEqual([words.vaultErrors.overlaps])
  })
})

describe('a vault renamed', () => {
  it('is called what was typed, and the window calls it that from now on', async () => {
    const one = window()

    await carry(invocationOf('renameVault', front(), 'Heat'), one.on)

    expect(one.done).toStrictEqual(['renames vault physics Heat', 'calls physics Heat'])
  })

  it('is left alone where the name it was given is the name it has', async () => {
    const one = window()

    await carry(invocationOf('renameVault', front(), 'Physics'), one.on)

    expect(one.done).toStrictEqual([])
  })

  it('says a name another vault is already called', async () => {
    const one = window({ added: asFailure('nameTaken') })

    await carry(invocationOf('renameVault', front(), 'Heat'), one.on)

    expect(one.said).toStrictEqual([words.vaultErrors.nameTaken])
  })
})

/** The vault taken off the list is the one that was chosen, never the one in front. */
describe('a vault taken off the list', () => {
  const heat = () => front({ vault: { id: 'heat', name: 'Heat' } })

  it('is forgotten, and its folder left where it is', async () => {
    const one = window()

    await carry(invocationOf('forgetVault', heat()), one.on)

    expect(one.done).toStrictEqual(['forgets heat'])
  })

  it('says the only vault this installation has stays on it', async () => {
    const one = window({ turnedDown: 'lastVault' })

    await carry(invocationOf('forgetVault', heat()), one.on)

    expect(one.said).toStrictEqual([words.vaultErrors.lastVault])
  })

  it('is erased where erasing was what was asked', async () => {
    const one = window()

    await carry(invocationOf('eraseVault', heat(), 'Heat'), one.on)

    expect(one.done).toStrictEqual(['erases heat'])
  })

  it('says a machine with nowhere to put what is deleted', async () => {
    const one = window({ turnedDown: 'noTrash' })

    await carry(invocationOf('eraseVault', heat(), 'Heat'), one.on)

    expect(one.said).toStrictEqual([words.vaultErrors.noTrash])
  })

  it('says the vault in front of the person, which the window stands on', async () => {
    const one = window({ turnedDown: 'showing' })

    await carry(invocationOf('forgetVault', front()), one.on)

    expect(one.said).toStrictEqual([words.vaultErrors.showing])
  })
})

describe('what the list of vaults reported as error', () => {
  it('reaches the person in the window’s own words, whichever it was', async () => {
    for (const err of Object.keys(words.vaultErrors) as VaultErrorCode[]) {
      const one = window({ turnedDown: err })

      await carry(invocationOf('openVault', front()), one.on)

      expect(words.vaultErrors[err], err).not.toBe('')
      expect(one.said, err).toStrictEqual([words.vaultErrors[err]])
    }
  })
})

describe('nothing to carry out', () => {
  it('does nothing at all', async () => {
    const one = window()

    await runInvocation(null, one.on, words)
    await carry(invocationOf('constructor', front()), one.on)
    await carry(invocationOf('', front()), one.on)

    expect(one.done).toStrictEqual([])
  })

  it('says what the vault could not be asked, and asks no further', async () => {
    const one = window()
    const broken: CommandDeps = {
      ...one.on,
      files: { ...one.on.files, remove: async () => Promise.reject(new Error('gone')) },
    }

    await runInvocation(invocationOf('remove', front()), broken, words)

    expect(one.said).toStrictEqual([
      'numen did not answer, so nothing was done — it may have stopped, and the window keeps trying',
    ])
  })
})

describe('the open files a command reaches', () => {
  /** One store, holding one file open at one path under one identity. */
  const store = (id: string, path: string, steps: string[]): Store => ({
    has: (one) => one === id,
    getPath: (one) => (one === id ? path : one),
    getTitle: (one) => (one === id ? `${id} called` : ''),
    isAsking: () => false,
    settle: async (one) => void steps.push(`settle ${one}`),
    close: (one) => void steps.push(`close ${one}`),
    getTabAt: (one) => (one === path ? id : null),
  })

  const createStores = () => {
    const done: string[] = []
    const stores = [store('note', 'Ontology.md', done), store('Animals.md', 'Animals.md', done)]
    return { done, notes: createNotes(stores, { openFile: () => {}, openNewFile: () => {} }) }
  }

  it('is the tab of whichever store stands at the file', () => {
    const one = createStores()

    expect(one.notes.getTabAt('Ontology.md')).toBe('note')
    expect(one.notes.getTabAt('Animals.md')).toBe('Animals.md')
    expect(one.notes.getTabAt('Loose.md')).toBeNull()
  })

  it('settles and shuts the store holding the identity, and no other', async () => {
    const one = createStores()

    await one.notes.settle('Animals.md')
    one.notes.close('Animals.md')

    expect(one.done).toStrictEqual(['settle Animals.md', 'close Animals.md'])
  })

  it('leaves an identity no store holds where it was, and does nothing to it', async () => {
    const one = createStores()

    await one.notes.settle('Gone.md')
    one.notes.close('Gone.md')

    expect(one.notes.getPath('Gone.md')).toBe('Gone.md')
    expect(one.notes.isAsking('Gone.md')).toBe(false)
    expect(one.done).toStrictEqual([])
  })
})
