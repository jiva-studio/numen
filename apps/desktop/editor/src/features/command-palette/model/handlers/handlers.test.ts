/**
 * Carrying a command out, asked without a window.
 *
 * Two things are asked here: that a note's pending write is flushed before its
 * the person.
 */
import { describe, expect, it } from 'vitest'
import { createCommands } from '../../lib/commands/commands'
import { runSupport } from '../palette/runSupport'
import { createInvocation } from '../../lib/offers/invocation'
import { createNotes } from '../../lib/notes/notes'
import type { VaultRef } from '../../types'
import type { CommandDeps } from './deps'
import type { CommandInvocation, CommandTarget, Store } from '../../types'
import { commands, flushTab, registerCommand, runInvocation } from './handlers'
import { asFailure, asValue } from '@numen/wire'
import type { Movement } from '@/entities/file'
import type { RemovedNote, RemoveResult, RenamedNote, RenameResult } from '@/entities/note'
import type { Vault, VaultErrorCode, VaultResult } from '@/entities/vault'
import type { ErrorCode } from '@/shared/errors'
import { WORDS } from '@/shared/words'
import { ANSWER_WORDS as words } from '../../words'

/** What is in front, which every invocation is carried out over. */
const front = (override: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: 'tab',
  kind: 'plex',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: '',
  source: null,
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
  ...override,
})

/** One vault as the list answers one. */
const createVault = (id: string, name: string): Vault => ({
  id,
  name,
  path: `/vaults/${name}`,
  isMissing: false,
})

const createRenameResult = (override: Partial<RenamedNote> = {}): RenameResult =>
  asValue({
    path: 'physics/Entropy.md',
    title: 'Entropy',
    hasFrontmatter: false,
    moved: null,
    ...override,
  })

const createRemoveResult = (override: Partial<RemovedNote> = {}): RemoveResult =>
  asValue({
    trashed: '.trash/Ontology.md',
    dangling: [],
    ...override,
  })

/**
 * A window that writes down everything a command asked of it, in order.
 *
 * before it moved can be carried out after.
 */
const window = (
  answers: {
    renamed?: RenameResult
    removed?: RemoveResult
    wasMade?: boolean
    /** Where the tab holding the note stands now. */
    targetPath?: string
    at?: string
    /** The note is waiting on the person, so nothing may move its file. */
    isAsking?: boolean
    /** The folder the person chose in the machine's own picker. */
    chose?: string
    /** What the list of vaults answered adding or renaming one. */
    added?: VaultResult
    turnedDown?: VaultErrorCode
    /** What moving a file came back with. */
    movement?: Movement
    /** What making a folder came back with. */
    folderError?: ErrorCode
  } = {},
) => {
  const isDone: string[] = []
  const notices: string[] = []
  /** The voice each sentence was said in, one to a sentence. */
  const tones: string[] = []
  /** Every call to the voice, the empty ones and the missing tones included. */
  const calls: [string | undefined, string | undefined][] = []
  const targetPath = answers.targetPath ?? answers.at ?? 'physics/Ontology.md'
  const turnedDown = answers.turnedDown ?? null
  /** The runs this one window has been told this build cannot do. */
  const runs = runSupport()
  const on: CommandDeps = {
    files: {
      createNote: async (title, from, relation) => {
        isDone.push(`createNote ${title} ${from || '—'} ${relation ?? '—'}`)
        return answers.wasMade === false ? null : { path: `${title}.md`, title }
      },
      rename: async (path, title) => {
        isDone.push(`rename ${path} ${title}`)
        return answers.renamed ?? createRenameResult()
      },
      remove: async (path, isPermanent) => {
        isDone.push(`remove ${path} ${isPermanent}`)
        return answers.removed ?? createRemoveResult()
      },
      move: async (from, to) => {
        isDone.push(`move ${from} ${to}`)
        return answers.movement ?? { ok: true, value: null }
      },
      createFolder: async (path) => {
        isDone.push(`createFolder ${path}`)
        return answers.folderError ?? null
      },
    },
    notes: {
      getTabAt: (path) => (path === targetPath ? 'active' : null),
      getPath: (id) => (id === 'active' || id === 'held' ? targetPath : id),
      asking: () => answers.isAsking === true,
      flushPending: async (id) => void isDone.push(`flushPending ${id}`),
      close: (id) => void isDone.push(`close ${id}`),
      openFile: (path, title, showing) => void isDone.push(`openFile ${path} ${title} ${showing}`),
      openNewFile: (path, title, options) => {
        const type = typeof options === 'string' ? options : options.type
        const showing = typeof options === 'object' ? options.where : undefined
        isDone.push(`openNewFile ${type} ${path} ${title} ${showing}`)
      },
    },
    vaults: {
      list: async () => ({ vaults: [createVault('physics', 'Physics')], showing: 'physics' }),
      choose: async (title) => {
        isDone.push(`choose ${title}`)
        return answers.chose ?? '/vaults/Heat'
      },
      add: async (path, name) => {
        isDone.push(`add ${path} ${name || '—'}`)
        return answers.added ?? asValue(createVault('heat', 'Heat'))
      },
      rename: async (id, name) => {
        isDone.push(`renames vault ${id} ${name}`)
        return answers.added ?? asValue(createVault(id, name))
      },
      remove: async (id, trash) => {
        isDone.push(trash ? `erases ${id}` : `removes ${id}`)
        return turnedDown
      },
      open: async (id) => {
        isDone.push(`opens vault ${id}`)
        return turnedDown
      },
      setVaultName: (vault: VaultRef) => void isDone.push(`calls ${vault.id} ${vault.name}`),
      reload: () => void isDone.push('reloads'),
    },
    navigator: {
      travel: async (path) => void isDone.push(`travel ${path}`),
      leave: async (from, to) => void isDone.push(`leave ${from} ${to}`),
      getOpeningNote: () => 'Root.md',
      openTab: (kind) => void isDone.push(`openTab ${kind}`),
      closeTab: (tab) => void isDone.push(`closeTab ${tab}`),
      search: () => void isDone.push('search'),
    },
    settings: {
      chooseAppearance: async (selected) => void isDone.push(`appearance ${selected}`),
      chooseSync: async (selected) => void isDone.push(`syncing ${selected}`),
      choosePerformance: async (selected) => void isDone.push(`performance ${selected}`),
      choosePluginSetting: async (pluginId, key, selected) =>
        void isDone.push(`plugin ${pluginId} ${key} ${selected}`),
    },
    runSupport: runs,
    copyPath: (path) => void isDone.push(`copies ${path}`),
    writeMessage: (text, kind) => {
      calls.push([text, kind])
      if (!text) return
      notices.push(text)
      tones.push(kind ?? '')
    },
  }
  return { on, actions: isDone, notices, tones, calls, runs }
}

/** One command carried out over the note in front. */
const carry = async (invocation: CommandInvocation, on: CommandDeps) =>
  runInvocation(invocation, on, words)

describe('every command that is offered', () => {
  it('is carried out by something', async () => {
    for (const command of createCommands(WORDS)) {
      const one = window()
      await carry(createInvocation(command.id, front(), 'Entropy'), one.on)

      expect(one.actions, command.id).not.toStrictEqual([])
    }
  })

  it('emits onCommand:<id> when a command is executed', async () => {
    const events: string[] = []
    const one = window()
    const onWithEmit: CommandDeps = {
      ...one.on,
      emitEvent: (ev) => events.push(ev),
    }

    await carry(createInvocation('read', front()), onWithEmit)

    expect(events).toStrictEqual(['onCommand:read'])
  })
})

describe('pluggable command registry', () => {
  it('registers dynamic commands and carries them out', async () => {
    const executed: string[] = []
    const unregister = registerCommand('customAction', (invocation) => {
      executed.push(`custom:${invocation.path}`)
    })

    const one = window()
    await carry(createInvocation('customAction', front()), one.on)
    expect(executed).toStrictEqual(['custom:physics/Ontology.md'])

    unregister()
    await carry(createInvocation('customAction', front()), one.on)
    expect(executed).toHaveLength(1)
  })

  it('allows overriding existing commands via commands.registerCommand', async () => {
    const executed: string[] = []
    const unregister = commands.registerCommand('read', () => {
      executed.push('custom-read')
    })

    const one = window()
    await carry(createInvocation('read', front()), one.on)
    expect(executed).toStrictEqual(['custom-read'])
    expect(one.actions).toStrictEqual([])

    unregister()
    await carry(createInvocation('read', front()), one.on)
    expect(one.actions).toContain('openFile physics/Ontology.md Ontology here')
  })
})

describe('a note put in front of the person', () => {
  it('opens in a tab of its own, and alongside it where the second key reached it', async () => {
    const one = window()

    await carry(createInvocation('read', front()), one.on)
    await carry(createInvocation('beside', front()), one.on)

    expect(one.actions).toStrictEqual([
      'openFile physics/Ontology.md Ontology here',
      'openFile physics/Ontology.md Ontology beside',
    ])
  })
})

describe('a note made', () => {
  it('writes the note it was made from into it, in the relation that was requested for', async () => {
    const one = window()

    await carry(createInvocation('child', front(), 'Entropy'), one.on)

    expect(one.actions[0]).toBe('createNote Entropy physics/Ontology.md child')
  })

  it('relies on its own where no relation was requested for', async () => {
    const one = window()

    await carry(createInvocation('note', front(), 'Entropy'), one.on)

    expect(one.actions[0]).toBe('createNote Entropy — —')
  })

  it('is put in front of the person', async () => {
    const one = window()

    await carry(createInvocation('child', front(), 'Entropy'), one.on)

    expect(one.actions.at(-1)).toBe('travel Entropy.md')
  })

  it('opens in a tab alongside the note the person is in', async () => {
    const one = window()

    await carry(createInvocation('child', front({ kind: 'note' }), 'Entropy'), one.on)

    expect(one.actions.at(-1)).toBe('openNewFile note Entropy.md Entropy beside')
  })

  it('takes the person nowhere where the vault would not make it', async () => {
    const one = window({ wasMade: false })

    await carry(createInvocation('child', front(), 'Entropy'), one.on)

    expect(one.actions).toStrictEqual(['createNote Entropy physics/Ontology.md child'])
  })

  it('is nothing at all where nothing was typed', async () => {
    const one = window()

    await carry(createInvocation('child', front(), ''), one.on)

    expect(one.actions).toStrictEqual([])
  })
})

describe('a note renamed', () => {
  it('has nothing on its way to its file before the file moves', async () => {
    const one = window()

    await carry(createInvocation('title', front(), 'Entropy'), one.on)

    expect(one.actions).toStrictEqual(['flushPending active', 'rename physics/Ontology.md Entropy'])
  })

  it('is rejected while the note is waiting on the person', async () => {
    const one = window({ isAsking: true })

    await carry(createInvocation('title', front(), 'Entropy'), one.on)

    expect(one.actions).toStrictEqual([])
    expect(one.notices).toStrictEqual([words.unanswered])
  })

  it('says the note was written elsewhere while this was requested', async () => {
    const one = window({ renamed: asFailure('changed' as const) })

    await carry(createInvocation('title', front(), 'Entropy'), one.on)

    expect(one.notices).toStrictEqual([words.stale])
  })

  it('is renamed where no tab of the window holds it', async () => {
    const one = window()

    await carry(createInvocation('title', front({ path: 'Elsewhere.md' }), 'Entropy'), one.on)

    expect(one.actions).toStrictEqual(['rename Elsewhere.md Entropy'])
  })

  it('is left alone where the name it was given is the name it has', async () => {
    const one = window()

    await carry(createInvocation('title', front(), 'Ontology'), one.on)

    expect(one.actions).toStrictEqual([])
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

    await carry(createInvocation('title', front(), 'Entropy'), one.on)

    expect(one.notices).toStrictEqual([])
  })

  it('says nothing of the title it wrote into the frontmatter', async () => {
    const one = window({ renamed: createRenameResult({ hasFrontmatter: true }) })

    await carry(createInvocation('title', front(), 'Entropy'), one.on)

    expect(one.notices).toStrictEqual([])
  })

  it('says the name was taken, and that the note carries the new one', async () => {
    const one = window({ renamed: asFailure('occupied' as const) })

    await carry(createInvocation('title', front(), 'Entropy'), one.on)

    expect(one.notices).toStrictEqual([words.errors.occupied])
  })

  it('says a note whose frontmatter cannot be read cannot be renamed', async () => {
    const one = window({ renamed: asFailure('unreadable' as const) })

    await carry(createInvocation('title', front(), 'Entropy'), one.on)

    expect(one.notices).toStrictEqual([words.errors.unreadable])
  })

  it('says a name no file can be named', async () => {
    const one = window({ renamed: asFailure('unnameable' as const) })

    await carry(createInvocation('title', front(), '...'), one.on)

    expect(one.notices).toStrictEqual([words.errors.unnameable])
  })
})

describe('a note removed', () => {
  it('has nothing on its way to its file before the file goes', async () => {
    const one = window()

    await carry(createInvocation('remove', front()), one.on)

    expect(one.actions.slice(0, 2)).toStrictEqual([
      'flushPending active',
      'remove physics/Ontology.md false',
    ])
  })

  it('goes off the disk where destroying was what was requested', async () => {
    const one = window()

    await carry(createInvocation('destroy', front(), 'Ontology'), one.on)

    expect(one.actions[1]).toBe('remove physics/Ontology.md true')
  })

  /** The row has gone from the tree, which is the whole of what a person needs. */
  it('says nothing about a note that went where it was requested to go', async () => {
    const one = window()

    await carry(createInvocation('remove', front()), one.on)

    expect(one.notices).toStrictEqual([])
  })

  it('says the notes that link to nothing now', async () => {
    const one = window({ removed: createRemoveResult({ dangling: ['Order.md', 'Notes.md'] }) })

    await carry(createInvocation('remove', front()), one.on)

    expect(one.notices).toStrictEqual([`${words.dangling} Order.md, Notes.md`])
  })

  it('leaves every plex standing on it at the note the vault opens with', async () => {
    const one = window()

    await carry(createInvocation('remove', front()), one.on)

    expect(one.actions.at(-1)).toBe('leave physics/Ontology.md Root.md')
  })

  it('leaves the plexes alone where the vault opens with no note at all', async () => {
    const one = window()
    const nowhere: CommandDeps = {
      ...one.on,
      navigator: { ...one.on.navigator, getOpeningNote: () => '' },
    }

    await carry(createInvocation('remove', front()), nowhere)

    expect(one.actions.some((step) => step.startsWith('leave'))).toBe(false)
  })

  it('lets go of the tab that was reading it', async () => {
    const one = window()

    await carry(createInvocation('remove', front(), '', 'active'), one.on)

    expect(one.actions).toContain('close active')
  })

  it('keeps the tab of a note the vault would not remove', async () => {
    const one = window({ removed: asFailure('missing' as const) })

    await carry(createInvocation('remove', front(), '', 'active'), one.on)

    expect(one.actions).not.toContain('close active')
  })

  it('says a note that is not in the vault, and takes the plex nowhere', async () => {
    const one = window({ removed: asFailure('missing' as const) })

    await carry(createInvocation('remove', front()), one.on)

    expect(one.notices).toStrictEqual([words.errors.missing])
    expect(one.actions.at(-1)).toBe('remove physics/Ontology.md false')
  })

  it('is rejected while the note is waiting on the person', async () => {
    const one = window({ isAsking: true })

    await carry(createInvocation('remove', front()), one.on)

    expect(one.actions).toStrictEqual([])
    expect(one.notices).toStrictEqual([words.unanswered])
  })
})

describe('several files removed at once', () => {
  const both = front({ others: ['physics/Heat.pdf'] })

  it('takes each of them out of the vault, in the order they were given', async () => {
    const one = window()

    await carry(createInvocation('remove', both), one.on)

    expect(one.actions.filter((step) => step.startsWith('remove'))).toStrictEqual([
      'remove physics/Ontology.md false',
      'remove physics/Heat.pdf false',
    ])
  })

  it('takes every plex standing on one of them to the note the vault opens with', async () => {
    const one = window()

    await carry(createInvocation('remove', both), one.on)

    expect(one.actions.filter((step) => step.startsWith('leave'))).toStrictEqual([
      'leave physics/Ontology.md Root.md',
      'leave physics/Heat.pdf Root.md',
    ])
  })

  it('says the notes that link to nothing now, each of them once', async () => {
    const one = window({ removed: createRemoveResult({ trashed: '', dangling: ['Order.md'] }) })

    await carry(createInvocation('remove', both), one.on)

    expect(one.notices).toStrictEqual([`${words.dangling} Order.md`])
  })

  it('takes the rest out where the vault refuses one, and says what failed', async () => {
    const one = window()
    const picky: CommandDeps = {
      ...one.on,
      files: {
        ...one.on.files,
        remove: async (path, destroy) => {
          one.actions.push(`remove ${path} ${destroy}`)
          return path === 'physics/Ontology.md'
            ? asFailure('missing' as const)
            : createRemoveResult()
        },
      },
    }

    await carry(createInvocation('remove', both), picky)

    expect(one.actions.filter((step) => step.startsWith('remove'))).toStrictEqual([
      'remove physics/Ontology.md false',
      'remove physics/Heat.pdf false',
    ])
    expect(one.notices).toStrictEqual([words.errors.missing])
  })
})

describe('a file filed somewhere else', () => {
  /** The destination is the whole path, so a name changed in one folder is a move. */
  const createMoveInvocation = (to: string) => createInvocation('move', front(), to)

  it('is requested of the vault under the path it is filed at from now on', async () => {
    const one = window()

    await carry(createMoveInvocation('notes/Ontology.md'), one.on)

    expect(one.actions).toStrictEqual([
      'flushPending active',
      'move physics/Ontology.md notes/Ontology.md',
    ])
  })

  it('flushes the tab holding it before its file goes anywhere', async () => {
    const one = window()

    await carry(createMoveInvocation('notes/Ontology.md'), one.on)

    expect(one.actions.indexOf('flushPending active')).toBeLessThan(
      one.actions.indexOf('move physics/Ontology.md notes/Ontology.md'),
    )
  })

  it('stays where it is while its tab is waiting on the person', async () => {
    const one = window({ isAsking: true })

    await carry(createMoveInvocation('notes/Ontology.md'), one.on)

    expect(one.actions).toStrictEqual([])
    expect(one.notices).toStrictEqual([words.unanswered])
  })

  it('stays where it is where something of that name is filed there', async () => {
    const one = window({ movement: { ok: false, error: 'occupied' } })

    await carry(createMoveInvocation('notes/Ontology.md'), one.on)

    expect(one.notices).toStrictEqual([words.occupied])
  })

  it('says nothing of a note renamed, which is what a move is not', async () => {
    const one = window({ movement: { ok: false, error: 'occupied' } })

    await carry(createMoveInvocation('notes/Ontology.md'), one.on)

    expect(one.notices).not.toContain(words.errors.occupied)
  })

  it('asks the vault for nothing where it landed where it already was', async () => {
    const one = window()

    await carry(createMoveInvocation('physics/Ontology.md'), one.on)

    expect(one.actions).toStrictEqual([])
  })
})

describe('a folder made', () => {
  it('is requested of the vault under the path it goes at', async () => {
    const one = window()

    await carry(createInvocation('createFolder', front(), 'physics/heat'), one.on)

    expect(one.actions).toStrictEqual(['createFolder physics/heat'])
  })

  it('is not made where something of that name is filed there', async () => {
    const one = window({ folderError: 'occupied' })

    await carry(createInvocation('createFolder', front(), 'physics/heat'), one.on)

    expect(one.notices).toStrictEqual([words.occupied])
  })

  it('asks the vault for nothing where no path was given', async () => {
    const one = window()

    await carry(createInvocation('createFolder', front()), one.on)

    expect(one.actions).toStrictEqual([])
  })
})

/**
 * An invocation is made when a person answers and carried out a moment later, and the
 * vault moves in between. The tab holding the note is what says where it is.
 */
describe('a note that moved between the answer and the invocation', () => {
  it('is renamed at its current position now, not at the name the invocation was made over', async () => {
    const one = window({ targetPath: 'physics/Being.md' })

    await carry(createInvocation('title', front(), 'Substance', 'held'), one.on)

    expect(one.actions).toStrictEqual(['flushPending active', 'rename physics/Being.md Substance'])
  })

  it('is removed at its current position now', async () => {
    const one = window({ targetPath: 'physics/Being.md' })

    await carry(createInvocation('remove', front(), '', 'active'), one.on)

    expect(one.actions.slice(0, 2)).toStrictEqual([
      'flushPending active',
      'remove physics/Being.md false',
    ])
  })

  it('is left at the name it was made over where no tab holds it', async () => {
    const one = window({ targetPath: 'physics/Being.md' })

    await carry(createInvocation('remove', front()), one.on)

    expect(one.actions[0]).toBe('remove physics/Ontology.md false')
  })
})

describe('a command over the window', () => {
  it('opens the settings, and closes the tab in front', async () => {
    const one = window()

    await carry(createInvocation('settings', front()), one.on)
    await carry(createInvocation('close', front()), one.on)

    expect(one.actions).toStrictEqual(['openTab settings', 'closeTab tab'])
  })

  it('passes the field back to the search', async () => {
    const one = window()

    await carry(createInvocation('find', front()), one.on)

    expect(one.actions).toStrictEqual(['search'])
  })

  it('passes over the row that was chosen, and nothing about the note in front', async () => {
    const one = window()

    await carry(createInvocation('appearance', front(), 'mine:sea'), one.on)
    await carry(createInvocation('appearance', front(), 'mode:dark'), one.on)

    expect(one.actions).toStrictEqual(['appearance mine:sea', 'appearance mode:dark'])
  })
})

describe('a command over the vault', () => {
  it('travels to the note the vault opens with', async () => {
    const one = window()

    await carry(createInvocation('first', front()), one.on)

    expect(one.actions).toStrictEqual(['travel Root.md'])
  })

  it('says a vault that opens with no note at all', async () => {
    const one = window()
    const empty: CommandDeps = {
      ...one.on,
      navigator: { ...one.on.navigator, getOpeningNote: () => '' },
    }

    await runInvocation(createInvocation('first', front()), empty, words)

    expect(one.notices).toStrictEqual([words.nowhere])
  })
})

describe('what the window is requested about a note', () => {
  it('has its path put on the clipboard', async () => {
    const one = window()

    await carry(createInvocation('copy', front()), one.on)

    expect(one.actions).toStrictEqual(['copies physics/Ontology.md'])
  })
})

describe('another vault under this window', () => {
  const heat = () => front({ vault: { id: 'heat', name: 'Heat' } })

  it('is opened, and the page drawn again on it', async () => {
    const one = window()

    await carry(createInvocation('openVault', heat()), one.on)

    expect(one.actions).toStrictEqual(['opens vault heat', 'reloads'])
  })

  it('leaves the page at its current position where the vault would not open', async () => {
    const one = window({ turnedDown: 'showing' })

    await carry(createInvocation('openVault', heat()), one.on)

    expect(one.actions).toStrictEqual(['opens vault heat'])
    expect(one.notices).toStrictEqual([words.vaultErrors.showing])
  })
})

describe('a vault made', () => {
  it('is the folder chosen in the machine’s own picker, and is opened', async () => {
    const one = window()

    await carry(createInvocation('newVault', front()), one.on)

    expect(one.actions).toStrictEqual([
      `choose ${words.folder}`,
      'add /vaults/Heat —',
      'opens vault heat',
      'reloads',
    ])
  })

  it('is nothing at all where the person closed the picker', async () => {
    const one = window({ chose: '' })

    await carry(createInvocation('newVault', front()), one.on)

    expect(one.actions).toStrictEqual([`choose ${words.folder}`])
    expect(one.notices).toStrictEqual([])
  })

  it('says a folder that lies inside a vault already added', async () => {
    const one = window({ added: asFailure('overlaps') })

    await carry(createInvocation('newVault', front()), one.on)

    expect(one.notices).toStrictEqual([words.vaultErrors.overlaps])
  })
})

describe('a vault renamed', () => {
  it('is called what was typed, and the window calls it that from now on', async () => {
    const one = window()

    await carry(createInvocation('renameVault', front(), 'Heat'), one.on)

    expect(one.actions).toStrictEqual(['renames vault physics Heat', 'calls physics Heat'])
  })

  it('is left alone where the name it was given is the name it has', async () => {
    const one = window()

    await carry(createInvocation('renameVault', front(), 'Physics'), one.on)

    expect(one.actions).toStrictEqual([])
  })

  it('says a name another vault is already called', async () => {
    const one = window({ added: asFailure('nameTaken') })

    await carry(createInvocation('renameVault', front(), 'Heat'), one.on)

    expect(one.notices).toStrictEqual([words.vaultErrors.nameTaken])
  })
})

/** The vault taken off the list is the one that was chosen, never the one in front. */
describe('a vault taken off the list', () => {
  const heat = () => front({ vault: { id: 'heat', name: 'Heat' } })

  it('is forgotten, and its folder left where it is', async () => {
    const one = window()

    await carry(createInvocation('removeVault', heat()), one.on)

    expect(one.actions).toStrictEqual(['removes heat'])
  })

  it('says the only vault this installation has stays on it', async () => {
    const one = window({ turnedDown: 'lastVault' })

    await carry(createInvocation('removeVault', heat()), one.on)

    expect(one.notices).toStrictEqual([words.vaultErrors.lastVault])
  })

  it('is erased where erasing was what was requested', async () => {
    const one = window()

    await carry(createInvocation('eraseVault', heat(), 'Heat'), one.on)

    expect(one.actions).toStrictEqual(['erases heat'])
  })

  it('says a machine with nowhere to put what is deleted', async () => {
    const one = window({ turnedDown: 'noTrash' })

    await carry(createInvocation('eraseVault', heat(), 'Heat'), one.on)

    expect(one.notices).toStrictEqual([words.vaultErrors.noTrash])
  })

  it('says the vault in front of the person, which the window relies on', async () => {
    const one = window({ turnedDown: 'showing' })

    await carry(createInvocation('removeVault', front()), one.on)

    expect(one.notices).toStrictEqual([words.vaultErrors.showing])
  })
})

describe('what the list of vaults reported as error', () => {
  it('reaches the person in the window’s own words, whichever it was', async () => {
    for (const err of Object.keys(words.vaultErrors) as VaultErrorCode[]) {
      const one = window({ turnedDown: err })

      await carry(createInvocation('openVault', front()), one.on)

      expect(words.vaultErrors[err], err).not.toBe('')
      expect(one.notices, err).toStrictEqual([words.vaultErrors[err]])
    }
  })
})

describe('nothing to carry out', () => {
  it('does nothing at all', async () => {
    const one = window()

    await runInvocation(null, one.on, words)
    await carry(createInvocation('constructor', front()), one.on)
    await carry(createInvocation('', front()), one.on)

    expect(one.actions).toStrictEqual([])
  })

  it('says what the vault could not be requested, and asks no further', async () => {
    const one = window()
    const broken: CommandDeps = {
      ...one.on,
      files: { ...one.on.files, remove: async () => Promise.reject(new Error('gone')) },
    }

    await runInvocation(createInvocation('remove', front()), broken, words)

    expect(one.notices).toStrictEqual(['Numen is not responding; retrying'])
  })
})

describe('the open files a command reaches', () => {
  /** One store, holding one file open at one path under one identity. */
  const store = (id: string, path: string, steps: string[]): Store => ({
    has: (one) => one === id,
    getPath: (one) => (one === id ? path : one),
    getTitle: (one) => (one === id ? `${id} called` : ''),
    asking: () => false,
    flushPending: async (one) => void steps.push(`flushPending ${one}`),
    close: (one) => void steps.push(`close ${one}`),
    getTabAt: (one) => (one === path ? id : null),
  })

  const createStores = () => {
    const isDone: string[] = []
    const stores = [store('note', 'Ontology.md', isDone), store('Animals.md', 'Animals.md', isDone)]
    return {
      actions: isDone,
      done: isDone,
      notes: createNotes(stores, { openFile: () => {}, openNewFile: () => {} }),
    }
  }

  it('is the tab of whichever store remains at the file', () => {
    const one = createStores()

    expect(one.notes.getTabAt('Ontology.md')).toBe('note')
    expect(one.notes.getTabAt('Animals.md')).toBe('Animals.md')
    expect(one.notes.getTabAt('Loose.md')).toBeNull()
  })

  it('flushes and shuts the store holding the identity, and no other', async () => {
    const one = createStores()

    await one.notes.flushPending('Animals.md')
    one.notes.close('Animals.md')

    expect(one.actions).toStrictEqual(['flushPending Animals.md', 'close Animals.md'])
  })

  it('leaves an identity no store holds where it was, and does nothing to it', async () => {
    const one = createStores()

    await one.notes.flushPending('Gone.md')
    one.notes.close('Gone.md')

    expect(one.notes.getPath('Gone.md')).toBe('Gone.md')
    expect(one.notes.asking('Gone.md')).toBe(false)
    expect(one.actions).toStrictEqual([])
  })
})

describe('a note made in a relation of another', () => {
  it('is made as a parent or as a jump where that was requested', async () => {
    const one = window()

    await carry(createInvocation('parent', front(), 'Entropy'), one.on)
    await carry(createInvocation('jump', front(), 'Heat'), one.on)

    expect(one.actions.filter((step) => step.startsWith('createNote'))).toStrictEqual([
      'createNote Entropy physics/Ontology.md parent',
      'createNote Heat physics/Ontology.md jump',
    ])
  })

  it('says nothing where the vault would not make it', async () => {
    const one = window({ wasMade: false })

    await carry(createInvocation('child', front(), 'Entropy'), one.on)

    expect(one.calls).toStrictEqual([['', undefined]])
  })
})

describe('a command nothing carries out', () => {
  it('does nothing and says nothing', async () => {
    const one = window()

    await carry({ ...createInvocation('read', front()), id: 'nothing' }, one.on)

    expect(one.actions).toStrictEqual([])
    expect(one.calls).toStrictEqual([])
  })
})

describe('what the voice says in which tone', () => {
  it('cautions that a vault opens with no note at all', async () => {
    const one = window()
    const empty: CommandDeps = {
      ...one.on,
      navigator: { ...one.on.navigator, getOpeningNote: () => '' },
    }

    await runInvocation(createInvocation('first', front()), empty, words)

    expect(one.tones).toStrictEqual(['caution'])
  })

  it('cautions that a rename waits on the person, and errors on failure', async () => {
    const pendingItem = window({ isAsking: true })
    const failedWindow = window({ renamed: asFailure('unnameable' as const) })
    const changedWindow = window({ renamed: asFailure('changed' as const) })

    await carry(createInvocation('title', front(), 'Entropy'), pendingItem.on)
    await carry(createInvocation('title', front(), 'Entropy'), failedWindow.on)
    await carry(createInvocation('title', front(), 'Entropy'), changedWindow.on)

    expect(pendingItem.tones).toStrictEqual(['caution'])
    expect(failedWindow.tones).toStrictEqual(['error'])
    expect(changedWindow.tones).toStrictEqual(['caution'])
  })

  it('says nothing of a rename that came off', async () => {
    const one = window()

    await carry(createInvocation('title', front(), 'Entropy'), one.on)

    expect(one.calls).toStrictEqual([['', undefined]])
  })

  it('cautions that a move waits on the person, and errors on failure', async () => {
    const pendingItem = window({ isAsking: true })
    const acquiredItem = window({ movement: { ok: false, error: 'occupied' } })
    const failingWindow = window({ movement: { ok: false, error: 'missing' } })

    await carry(createInvocation('move', front(), 'notes/Ontology.md'), pendingItem.on)
    await carry(createInvocation('move', front(), 'notes/Ontology.md'), acquiredItem.on)
    await carry(createInvocation('move', front(), 'notes/Ontology.md'), failingWindow.on)

    expect(pendingItem.tones).toStrictEqual(['caution'])
    expect(acquiredItem.tones).toStrictEqual(['error'])
    expect(failingWindow.notices).toStrictEqual([words.errors.missing])
    expect(failingWindow.tones).toStrictEqual(['error'])
  })

  it('says nothing of a move that came off', async () => {
    const one = window()

    await carry(createInvocation('move', front(), 'notes/Ontology.md'), one.on)

    expect(one.calls).toStrictEqual([['', undefined]])
  })

  it('errors on a folder that is taken, and says what else rejected it', async () => {
    const acquiredItem = window({ folderError: 'occupied' })
    const failingWindow = window({ folderError: 'unnameable' })

    await carry(createInvocation('createFolder', front(), 'physics/heat'), acquiredItem.on)
    await carry(createInvocation('createFolder', front(), 'physics/heat'), failingWindow.on)

    expect(acquiredItem.tones).toStrictEqual(['error'])
    expect(failingWindow.notices).toStrictEqual([words.errors.unnameable])
    expect(failingWindow.tones).toStrictEqual(['error'])
  })

  it('says nothing of a folder that was made', async () => {
    const one = window()

    await carry(createInvocation('createFolder', front(), 'physics/heat'), one.on)

    expect(one.calls).toStrictEqual([['', undefined]])
  })

  it('errors on a removal the vault rejected, and cautions on one that waits', async () => {
    const failingWindow = window({ removed: asFailure('missing' as const) })
    const pendingItem = window({ isAsking: true })

    await carry(createInvocation('remove', front()), failingWindow.on)
    await carry(createInvocation('remove', front()), pendingItem.on)

    expect(failingWindow.tones).toStrictEqual(['error'])
    expect(pendingItem.notices).toStrictEqual([words.unanswered])
    expect(pendingItem.tones).toStrictEqual(['caution'])
  })
})

describe('a file removed that no tab holds', () => {
  it('closes no tab', async () => {
    const one = window()

    await carry(createInvocation('remove', front({ path: 'physics/Heat.pdf' })), one.on)

    expect(one.actions).toStrictEqual([
      'remove physics/Heat.pdf false',
      'leave physics/Heat.pdf Root.md',
    ])
  })
})

describe('the tab holding a note flushed', () => {
  it('is none, and not waiting, where no tab holds it', async () => {
    const one = window()

    expect(await flushTab('physics/Heat.pdf', one.on)).toStrictEqual({
      tabId: null,
      isWaiting: false,
    })
  })

  it('is the tab, flushed, where one holds it', async () => {
    const one = window()

    expect(await flushTab('physics/Ontology.md', one.on)).toStrictEqual({
      tabId: 'active',
      isWaiting: false,
    })
    expect(one.actions).toStrictEqual(['flushPending active'])
  })

  it('is the tab, waiting and not flushed, where it waits on the person', async () => {
    const one = window({ isAsking: true })

    expect(await flushTab('physics/Ontology.md', one.on)).toStrictEqual({
      tabId: 'active',
      isWaiting: true,
    })
    expect(one.actions).toStrictEqual([])
  })
})

describe('plugin setting invocation', () => {
  it('calls choosePluginSetting with boolean and enum values', async () => {
    const one = window()

    await carry(
      createInvocation('plugin-setting:md.numen.engines.transcription:autoRun', front(), 'on'),
      one.on,
    )
    expect(one.actions).toStrictEqual(['plugin md.numen.engines.transcription autoRun true'])

    await carry(
      createInvocation('plugin-setting:md.numen.engines.transcription:autoRun', front(), 'off'),
      one.on,
    )
    expect(one.actions[1]).toBe('plugin md.numen.engines.transcription autoRun false')

    await carry(
      createInvocation('plugin-setting:md.numen.engines.transcription:language', front(), 'en'),
      one.on,
    )
    expect(one.actions[2]).toBe('plugin md.numen.engines.transcription language en')
  })
})
