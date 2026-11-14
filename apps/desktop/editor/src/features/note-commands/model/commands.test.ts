/**
 * The commands over a note, carried out over the real calls of the entities and a double of the
 * window's tabs: what they offer, and what each one asks of the core and of the window.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { NAME_SOURCE, NOTE_ERROR_CODE, NOTE_ROLE } from '@numen/core-client'
import { installCoreHandlers } from '@numen/core-client/testing'
import { createNoteWriter } from '@/entities/note'
import type { CommandInvocation, CommandTarget } from '@/shared/model/command'
import { invoke } from '@/testing/invocation'
import { NOTE_WORDS as words } from '../words'
import { createNoteCommands } from './commands'

const front = (override: Partial<CommandTarget> = {}): CommandTarget => ({
  tab: 'tab',
  kind: 'plex',
  path: 'physics/Ontology.md',
  title: 'Ontology',
  file: '',
  vault: { id: 'physics', name: 'Physics' },
  isReady: true,
  ...override,
})

interface Answers {
  readonly rename?: object
  readonly remove?: object
  readonly isMade?: boolean
  readonly isAsking?: boolean
  readonly opening?: string
}

const restores: (() => void)[] = []

afterEach(() => {
  for (const restore of restores.splice(0)) restore()
  vi.unstubAllGlobals()
})

const createWindow = (answers: Answers = {}) => {
  const actions: string[] = []
  const notices: string[] = []
  const severities: string[] = []
  const targetPath = 'physics/Ontology.md'
  const writeMessage = (text: string, severity?: string) => {
    if (!text) return
    notices.push(text)
    severities.push(severity ?? '')
  }
  restores.push(
    installCoreHandlers({
      createNote: (request: { title: string; path: string }) => {
        actions.push(`createNote ${request.title || '—'} ${request.path || '—'}`)
        if (answers.isMade === false) return { isNameExhausted: true }
        return {
          path: request.title
            ? `${request.title}.md`
            : `${request.path ? `${request.path}/` : ''}Untitled note 1.md`,
        }
      },
      createRelatedNote: (request: { from: string; role: number; title: string }) => {
        const role = Object.entries(NOTE_ROLE)
          .find(([, code]) => code === request.role)?.[0]
          ?.toLowerCase()
        actions.push(`createRelated ${request.title} ${request.from} ${role}`)
        return answers.isMade === false
          ? { isNameExhausted: true }
          : { path: `${request.title}.md` }
      },
      renameNote: (request: { path: string; title: string }) => {
        actions.push(`rename ${request.path} ${request.title}`)
        return (
          answers.rename ?? {
            path: 'physics/Entropy.md',
            title: 'Entropy',
            nameSource: NAME_SOURCE.FILENAME,
          }
        )
      },
      listFileKinds: (request: { paths: string[] }) => ({
        kinds: request.paths.map((path) => ({ path, isNote: true, type: '', tabKind: 'page' })),
      }),
      removeFile: (request: { path: string }) => {
        actions.push(`remove ${request.path}`)
        return answers.remove ?? { trashed: '.trash/Ontology.md', dangling: [] }
      },
    }),
  )
  const deps = {
    making: createNoteWriter(writeMessage),
    notes: {
      getTabAt: (path: string) => (path === targetPath ? 'active' : null),
      getPath: (id: string) => id,
      asking: () => answers.isAsking === true,
      flushPending: async (id: string) => {
        actions.push(`flushPending ${id}`)
      },
      close: (id: string) => {
        actions.push(`close ${id}`)
      },
      openFile: (path: string, title: string, placement: string, place?: object) => {
        actions.push(
          `openFile ${path} ${title} ${placement}${place ? ` ${JSON.stringify(place)}` : ''}`,
        )
      },
    },
    navigator: {
      travel: async (path: string) => {
        actions.push(`travel ${path}`)
      },
      leave: async (from: string, to: string) => {
        actions.push(`leave ${from} ${to}`)
      },
      getOpeningNote: () => answers.opening ?? 'Root.md',
      showTab: (kind: string, path: string) => {
        actions.push(`showTab ${kind} ${path}`)
      },
    },
    writeMessage,
  }
  const set = createNoteCommands(deps, 'linux')
  const carry = async (invocation: CommandInvocation): Promise<string> => {
    const run = set.handlers[invocation.id]
    return (await run?.(invocation)) || ''
  }
  return { actions, notices, severities, carry, set }
}

describe('the commands over a note', () => {
  it('are all filed with the note commands, but the one that makes a note', () => {
    const { set } = createWindow()

    expect(set.commands.filter((one) => one.group !== 'note').map((one) => one.id)).toStrictEqual([
      'app:note.create',
    ])
  })

  it('ask for a name where they take one', () => {
    const { set } = createWindow()
    const inputs = Object.fromEntries(set.commands.map((one) => [one.id, one.input]))

    expect(inputs).toStrictEqual({
      'app:file.open': undefined,
      'app:file.open#beside': undefined,
      'app:file.reveal': undefined,
      'app:note.createChild': 'name',
      'app:note.createParent': 'name',
      'app:note.createJump': 'name',
      'app:note.rename': 'name',
      'app:file.remove': undefined,
      'app:file.copyPath': undefined,
      'app:note.create': 'name',
    })
  })

  it('open the note in the current group or a split by an argument of one command', () => {
    const { set } = createWindow()
    const find = (id: string) => set.commands.find((one) => one.id === id)

    expect(find('app:file.open')).toMatchObject({ also: 'app:file.open#beside' })
    expect(find('app:file.open#beside')).toMatchObject({
      command: 'app:file.open',
      values: { placement: 'split' },
    })
  })

  it('remove the note to the trash by a command with no argument and no further row', () => {
    const { set } = createWindow()
    const removing = set.commands.find((one) => one.id === 'app:file.remove')

    expect(removing).toBeDefined()
    expect(removing).not.toHaveProperty('also')
    expect(removing).not.toHaveProperty('input')
    expect(set.commands.map((one) => one.id)).not.toContain('app:file.remove#permanent')
  })

  it('is offered over every file but not over the vault root', () => {
    const { set } = createWindow()
    const removing = set.commands.find((one) => one.id === 'app:file.remove')
    const runs = { canRun: () => true, cannotRun: () => undefined }
    const over = (path: string) => removing?.isAvailable(front({ path, file: path }), runs) ?? false

    expect(over('physics/Ontology.md')).toBe(true)
    expect(over('assets/shore.pdf')).toBe(true)
    expect(over('assets/sample.epub')).toBe(true)
    expect(over('assets/lecture.wav')).toBe(true)
    expect(over('')).toBe(false)
  })

  it('hold no command of the old ids', () => {
    const { set } = createWindow()
    const ids = set.commands.map((one) => one.command ?? one.id)

    expect(ids).not.toContain('app:note.open')
    expect(ids).not.toContain('app:note.openBeside')
    expect(ids).not.toContain('app:note.trash')
    expect(ids).not.toContain('app:note.delete')
  })

  it('carry out every command that has a row of its own, but the one a slot draws', () => {
    const { set } = createWindow()
    const own = set.commands.filter((one) => one.command === undefined)

    for (const command of own) {
      if (command.id === 'app:file.reveal') continue
      expect(set.handlers[command.id], command.id).toBeTypeOf('function')
    }
  })
})

describe('a note put in front of the person', () => {
  it('opens in a tab of its own, and alongside it where the second key reached it', async () => {
    const one = createWindow()

    await one.carry(invoke('app:file.open', front()))
    await one.carry({ ...invoke('app:file.open', front()), values: { placement: 'split' } })

    expect(one.actions).toStrictEqual([
      'openFile physics/Ontology.md Ontology current',
      'openFile physics/Ontology.md Ontology split',
    ])
  })

  it('opens at the line of the target, or from the start where it has none', async () => {
    const one = createWindow()

    await one.carry(invoke('app:file.open', front({ line: 7 })))
    await one.carry(invoke('app:file.open', front({ line: 0 })))
    await one.carry(invoke('app:file.open', front({ selector: { page: 3 } })))

    expect(one.actions).toStrictEqual([
      'openFile physics/Ontology.md Ontology current {"line":7}',
      'openFile physics/Ontology.md Ontology current {"line":0}',
      'openFile physics/Ontology.md Ontology current {"selector":{"page":3}}',
    ])
  })

  it('names the tab by the path of the file where the file has no title', async () => {
    const one = createWindow()

    await one.carry(invoke('app:file.open', front({ title: '' })))

    expect(one.actions).toStrictEqual(['openFile physics/Ontology.md physics/Ontology.md current'])
  })
})

describe('a note shown in a place', () => {
  it('shows the note in the tab kind the command names, and does nothing without one', async () => {
    const one = createWindow()

    await one.carry({ ...invoke('app:file.reveal', front()), values: { kind: 'plex' } })
    await one.carry({ ...invoke('app:file.reveal', front()), values: { kind: 'files' } })
    await one.carry(invoke('app:file.reveal', front()))

    expect(one.actions).toStrictEqual([
      'showTab plex physics/Ontology.md',
      'showTab files physics/Ontology.md',
    ])
  })
})

describe('a note made', () => {
  it('is made from the note it was requested over, in the relation that was requested', async () => {
    const one = createWindow()
    const row = front({ tab: '', kind: null, path: 'reef/Coral.md', title: 'Coral', file: '' })

    await one.carry(invoke('app:note.createChild', row, 'Polyp'))
    await one.carry(invoke('app:note.createParent', row, 'Atoll'))
    await one.carry(invoke('app:note.createJump', row, 'Lagoon'))

    expect(one.actions.filter((action) => action.startsWith('createRelated'))).toStrictEqual([
      'createRelated Polyp reef/Coral.md child',
      'createRelated Atoll reef/Coral.md parent',
      'createRelated Lagoon reef/Coral.md jump',
    ])
  })

  it('relates to no note where no relation was requested', async () => {
    const one = createWindow()

    await one.carry(invoke('app:note.create', front(), 'Entropy'))

    expect(one.actions[0]).toBe('createNote Entropy —')
  })

  it('is put in front of the person', async () => {
    const one = createWindow()

    await one.carry(invoke('app:note.createChild', front(), 'Entropy'))

    expect(one.actions.at(-1)).toBe('travel Entropy.md')
  })

  it('opens the note in a split next to the current tab when that tab kind opens the note', async () => {
    const one = createWindow()

    await one.carry(invoke('app:note.createChild', front({ kind: 'page' }), 'Entropy'))

    expect(one.actions.at(-1)).toBe('openFile Entropy.md Entropy split')
  })

  it('takes the person nowhere where the vault would not make it', async () => {
    const one = createWindow({ isMade: false })

    await one.carry(invoke('app:note.createChild', front(), 'Entropy'))

    expect(one.actions).toStrictEqual(['createRelated Entropy physics/Ontology.md child'])
  })

  it('is nothing at all where nothing was typed', async () => {
    const one = createWindow()

    await one.carry(invoke('app:note.createChild', front(), ''))

    expect(one.actions).toStrictEqual([])
  })

  it('answers the path of a note made under a typed name', async () => {
    const one = createWindow()

    expect(await one.carry(invoke('app:note.create', front(), 'Entropy'))).toBe('Entropy.md')
    expect(await one.carry(invoke('app:note.createChild', front(), 'Heat'))).toBe('Heat.md')
  })

  it('answers nothing over a note without a name', async () => {
    const one = createWindow()

    expect(await one.carry(invoke('app:note.create', front()))).toBe('')
    expect(one.actions).toStrictEqual([])
  })
})

describe('a note made over a folder', () => {
  const folder = (path: string) => front({ path, file: path, isFolder: true, isTargetNamed: true })

  it('is untitled, filed in the folder, and answers its path', async () => {
    const one = createWindow()

    const made = await one.carry(invoke('app:note.create', folder('physics')))

    expect(made).toBe('physics/Untitled note 1.md')
    expect(one.actions).toStrictEqual(['createNote — physics'])
  })

  it('is filed at the root where the folder is the root', async () => {
    const one = createWindow()

    const made = await one.carry(invoke('app:note.create', folder('')))

    expect(made).toBe('Untitled note 1.md')
    expect(one.actions).toStrictEqual(['createNote — —'])
  })

  it('answers nothing where the vault made no note', async () => {
    const one = createWindow({ isMade: false })

    expect(await one.carry(invoke('app:note.create', folder('physics')))).toBe('')
  })

  it('keeps the typed name when one is given', async () => {
    const one = createWindow()

    const made = await one.carry(invoke('app:note.create', folder('physics'), 'Entropy'))

    expect(made).toBe('Entropy.md')
    expect(one.actions[0]).toBe('createNote Entropy —')
  })
})

describe('a note renamed', () => {
  it('has nothing on its way to its file before the file moves', async () => {
    const one = createWindow()

    await one.carry(invoke('app:note.rename', front(), 'Entropy'))

    expect(one.actions).toStrictEqual(['flushPending active', 'rename physics/Ontology.md Entropy'])
  })

  it('is rejected while the note is waiting on the person', async () => {
    const one = createWindow({ isAsking: true })

    await one.carry(invoke('app:note.rename', front(), 'Entropy'))

    expect(one.actions).toStrictEqual([])
    expect(one.notices).toStrictEqual([words.unanswered])
  })

  it('says the note was written elsewhere while this was requested', async () => {
    const one = createWindow({ rename: { error: NOTE_ERROR_CODE.CONFLICT } })

    await one.carry(invoke('app:note.rename', front(), 'Entropy'))

    expect(one.notices).toStrictEqual([words.stale])
  })

  it('is renamed where no tab of the window holds it', async () => {
    const one = createWindow()

    await one.carry(invoke('app:note.rename', front({ path: 'Elsewhere.md' }), 'Entropy'))

    expect(one.actions).toStrictEqual(['rename Elsewhere.md Entropy'])
  })

  it('is left alone where the name it was given is the name it has', async () => {
    const one = createWindow()

    await one.carry(invoke('app:note.rename', front(), 'Ontology'))

    expect(one.actions).toStrictEqual([])
  })

  it('says nothing of the notes whose links it wrote again', async () => {
    const one = createWindow({
      rename: {
        path: 'physics/Entropy.md',
        title: 'Entropy',
        nameSource: NAME_SOURCE.FILENAME,
        moved: {
          from: 'physics/Ontology.md',
          to: 'physics/Entropy.md',
          repaired: ['Notes.md', 'Order.md'],
        },
      },
    })

    await one.carry(invoke('app:note.rename', front(), 'Entropy'))

    expect(one.notices).toStrictEqual([])
  })

  it('says nothing of the title it wrote into the frontmatter', async () => {
    const one = createWindow({
      rename: { path: 'physics/Entropy.md', title: 'Entropy', nameSource: NAME_SOURCE.FRONTMATTER },
    })

    await one.carry(invoke('app:note.rename', front(), 'Entropy'))

    expect(one.notices).toStrictEqual([])
  })

  it('shows a name-taken error, and the note keeps its name', async () => {
    const one = createWindow({ rename: { error: NOTE_ERROR_CODE.ALREADY_EXISTS } })

    await one.carry(invoke('app:note.rename', front(), 'Entropy'))

    expect(one.notices).toStrictEqual([words.errors.alreadyExists])
  })

  it('says a note whose frontmatter cannot be read cannot be renamed', async () => {
    const one = createWindow({ rename: { error: NOTE_ERROR_CODE.UNREADABLE } })

    await one.carry(invoke('app:note.rename', front(), 'Entropy'))

    expect(one.notices).toStrictEqual([words.errors.unreadable])
  })

  it('says a name no file can be named', async () => {
    const one = createWindow({ rename: { error: NOTE_ERROR_CODE.INVALID_NAME } })

    await one.carry(invoke('app:note.rename', front(), '...'))

    expect(one.notices).toStrictEqual([words.errors.invalidName])
  })
})

describe('a note removed', () => {
  it('has nothing on its way to its file before the file is removed', async () => {
    const one = createWindow()

    await one.carry(invoke('app:file.remove', front()))

    expect(one.actions.slice(0, 2)).toStrictEqual([
      'flushPending active',
      'remove physics/Ontology.md',
    ])
  })

  it('says nothing about a note that went where it was requested to go', async () => {
    const one = createWindow()

    await one.carry(invoke('app:file.remove', front()))

    expect(one.notices).toStrictEqual([])
  })

  it('says the notes that link to nothing now', async () => {
    const one = createWindow({
      remove: { trashed: '.trash/Ontology.md', dangling: ['Order.md', 'Notes.md'] },
    })

    await one.carry(invoke('app:file.remove', front()))

    expect(one.notices).toStrictEqual([`${words.dangling} Order.md, Notes.md`])
    expect(one.severities).toStrictEqual(['warning'])
  })

  it('leaves every plex standing on it at the note the vault opens with', async () => {
    const one = createWindow()

    await one.carry(invoke('app:file.remove', front()))

    expect(one.actions.at(-1)).toBe('leave physics/Ontology.md Root.md')
  })

  it('leaves the plexes alone where the vault opens with no note at all', async () => {
    const one = createWindow({ opening: '' })

    await one.carry(invoke('app:file.remove', front()))

    expect(one.actions.some((step) => step.startsWith('leave'))).toBe(false)
  })

  it('releases the tab that was reading it', async () => {
    const one = createWindow()

    await one.carry(invoke('app:file.remove', front(), '', 'active'))

    expect(one.actions).toContain('close active')
  })

  it('keeps the tab of a note the vault would not remove', async () => {
    const one = createWindow({ remove: { error: NOTE_ERROR_CODE.NOT_FOUND } })

    await one.carry(invoke('app:file.remove', front(), '', 'active'))

    expect(one.actions).not.toContain('close active')
  })

  it('says a note that is not in the vault, and takes the plex nowhere', async () => {
    const one = createWindow({ remove: { error: NOTE_ERROR_CODE.NOT_FOUND } })

    await one.carry(invoke('app:file.remove', front()))

    expect(one.notices).toStrictEqual([words.errors.notFound])
    expect(one.actions.at(-1)).toBe('remove physics/Ontology.md')
  })

  it('is rejected while the note is waiting on the person', async () => {
    const one = createWindow({ isAsking: true })

    await one.carry(invoke('app:file.remove', front()))

    expect(one.actions).toStrictEqual([])
    expect(one.notices).toStrictEqual([words.unanswered])
  })
})

describe('several files removed at once', () => {
  const both = front({ others: ['physics/Heat.pdf'] })

  it('takes each of them out of the vault, in the order they were given', async () => {
    const one = createWindow()

    await one.carry(invoke('app:file.remove', both))

    expect(one.actions.filter((step) => step.startsWith('remove'))).toStrictEqual([
      'remove physics/Ontology.md',
      'remove physics/Heat.pdf',
    ])
  })

  it('takes every plex standing on one of them to the note the vault opens with', async () => {
    const one = createWindow()

    await one.carry(invoke('app:file.remove', both))

    expect(one.actions.filter((step) => step.startsWith('leave'))).toStrictEqual([
      'leave physics/Ontology.md Root.md',
      'leave physics/Heat.pdf Root.md',
    ])
  })

  it('says the notes that link to nothing now, each of them once', async () => {
    const one = createWindow({ remove: { trashed: '', dangling: ['Order.md'] } })

    await one.carry(invoke('app:file.remove', both))

    expect(one.notices).toStrictEqual([`${words.dangling} Order.md`])
  })
})

describe('the path of a note copied', () => {
  it('goes to the clipboard', async () => {
    const writeText = vi.fn(() => Promise.resolve())
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const one = createWindow()

    await one.carry(invoke('app:file.copyPath', front()))

    expect(writeText).toHaveBeenCalledExactlyOnceWith('physics/Ontology.md')
  })

  it('reports a clipboard that refuses it', async () => {
    vi.stubGlobal('navigator', {
      clipboard: { writeText: () => Promise.reject(new Error('denied')) },
    })
    const one = createWindow()

    await one.carry(invoke('app:file.copyPath', front()))
    await new Promise((isDone) => setTimeout(isDone, 0))

    expect(one.severities).toStrictEqual(['error'])
  })

  it('is nothing where there is no clipboard', async () => {
    vi.stubGlobal('navigator', {})
    const one = createWindow()

    await expect(one.carry(invoke('app:file.copyPath', front()))).resolves.toBe('')
    expect(one.notices).toStrictEqual([])
  })
})
