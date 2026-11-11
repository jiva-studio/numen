/**
 * The commands over files and folders, carried out over the real calls of the file entity and a
 * double of the window's tabs.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { NOTE_ERROR_CODE } from '@numen/core-client'
import { installCoreHandlers } from '@numen/core-client/testing'
import type { CommandInvocation, CommandTarget } from '@/shared/model/command'
import { invoke } from '@/testing/invocation'
import { FILE_WORDS as words } from '../words'
import { createFileCommands } from './commands'

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
  readonly move?: object
  readonly folder?: object
  readonly names?: readonly string[]
  readonly isAsking?: boolean
}

const restores: (() => void)[] = []

afterEach(() => {
  for (const restore of restores.splice(0)) restore()
})

const createWindow = (answers: Answers = {}) => {
  const actions: string[] = []
  const notices: string[] = []
  restores.push(
    installCoreHandlers({
      moveFile: (request: { from: string; to: string }) => {
        actions.push(`move ${request.from} ${request.to}`)
        return (
          answers.move ?? { moved: { from: request.from, to: request.to, repaired: [] } }
        )
      },
      listFiles: (request: { path: string }) => {
        actions.push(`listNames ${request.path || '—'}`)
        return {
          entries: (answers.names ?? []).map((name) => ({
            path: `${request.path}/${name}`,
            name,
            isFolder: true,
            isNote: false,
            type: '',
          })),
        }
      },
      createFolder: (request: { path: string }) => {
        actions.push(`createFolder ${request.path}`)
        return answers.folder ?? {}
      },
    }),
  )
  const deps = {
    notes: {
      getTabAt: (path: string) => (path === 'physics/Ontology.md' ? 'active' : null),
      getPath: (id: string) => id,
      asking: () => answers.isAsking === true,
      flushPending: async (id: string) => {
        actions.push(`flushPending ${id}`)
      },
      close: () => undefined,
      openFile: () => undefined,
    },
    writeMessage: (text: string) => {
      if (text) notices.push(text)
    },
  }
  const set = createFileCommands(deps)
  const carry = async (invocation: CommandInvocation): Promise<string> =>
    (await set.handlers[invocation.id]?.(invocation)) || ''
  return { actions, notices, carry, set }
}

describe('the commands over files', () => {
  it('offer a folder to make and a document to open, and no way to move a file', () => {
    const { set } = createWindow()

    expect(set.commands.map((one) => one.id)).toStrictEqual([
      'app:folder.create',
      'app:file.open#document',
    ])
  })

  it('carry out the commands that make a folder and move a file', () => {
    const { set } = createWindow()

    expect(Object.keys(set.handlers).sort()).toStrictEqual(['app:file.move', 'app:folder.create'])
  })

  it('open a document by running the command that opens a file', () => {
    const { set } = createWindow()

    expect(set.commands.find((one) => one.id === 'app:file.open#document')).toMatchObject({
      command: 'app:file.open',
      group: 'file',
      text: words.readDocument,
    })
  })
})

describe('a file filed somewhere else', () => {
  /** The destination is the whole path, so a name changed in one folder is a move. */
  const createMoveInvocation = (to: string) => invoke('app:file.move', front(), to)

  it('is requested of the vault under the path it is filed at from now on', async () => {
    const one = createWindow()

    await one.carry(createMoveInvocation('notes/Ontology.md'))

    expect(one.actions).toStrictEqual([
      'flushPending active',
      'move physics/Ontology.md notes/Ontology.md',
    ])
  })

  it('keeps its position while its tab is waiting on the person', async () => {
    const one = createWindow({ isAsking: true })

    await one.carry(createMoveInvocation('notes/Ontology.md'))

    expect(one.actions).toStrictEqual([])
    expect(one.notices).toStrictEqual([words.unanswered])
  })

  it('keeps its position where something of that name is filed there', async () => {
    const one = createWindow({ move: { error: NOTE_ERROR_CODE.ALREADY_EXISTS } })

    await one.carry(createMoveInvocation('notes/Ontology.md'))

    expect(one.notices).toStrictEqual([words.alreadyExists])
  })

  it('says why the vault refused it, in the words of the error', async () => {
    const one = createWindow({ move: { error: NOTE_ERROR_CODE.NOT_FOUND } })

    await one.carry(createMoveInvocation('notes/Ontology.md'))

    expect(one.notices).toStrictEqual([words.errors.notFound])
  })

  it('asks the vault for nothing where it landed where it already was', async () => {
    const one = createWindow()

    await one.carry(createMoveInvocation('physics/Ontology.md'))

    expect(one.actions).toStrictEqual([])
  })
})

describe('a folder made', () => {
  it('is requested of the vault under the path it goes at, and answers that path', async () => {
    const one = createWindow()

    const made = await one.carry(invoke('app:folder.create', front(), 'physics/heat'))

    expect(made).toBe('physics/heat')
    expect(one.actions).toStrictEqual(['createFolder physics/heat'])
  })

  it('is not made where something of that name is filed there', async () => {
    const one = createWindow({ folder: { error: NOTE_ERROR_CODE.ALREADY_EXISTS } })

    await one.carry(invoke('app:folder.create', front(), 'physics/heat'))

    expect(one.notices).toStrictEqual([words.alreadyExists])
  })

  it('asks the vault for nothing where no path was given', async () => {
    const one = createWindow()

    await one.carry(invoke('app:folder.create', front()))

    expect(one.actions).toStrictEqual([])
  })
})

describe('a folder made over a folder', () => {
  const folder = (path: string) => front({ path, file: path, isFolder: true, isTargetNamed: true })

  it('is named in the folder, filed there, and answers its path', async () => {
    const one = createWindow()

    const made = await one.carry(invoke('app:folder.create', folder('physics')))

    expect(made).toBe('physics/New folder')
    expect(one.actions).toStrictEqual(['listNames physics', 'createFolder physics/New folder'])
  })

  it('is filed at the root where the folder is the root', async () => {
    const one = createWindow()

    const made = await one.carry(invoke('app:folder.create', folder('')))

    expect(made).toBe('New folder')
    expect(one.actions).toStrictEqual(['listNames —', 'createFolder New folder'])
  })

  it('counts past the names the folder already holds', async () => {
    const one = createWindow({ names: ['New folder', 'New folder 2'] })

    const made = await one.carry(invoke('app:folder.create', folder('physics')))

    expect(made).toBe('physics/New folder 3')
  })

  it('is made under the typed path where one was typed', async () => {
    const one = createWindow()

    const made = await one.carry(invoke('app:folder.create', folder('physics'), 'optics'))

    expect(made).toBe('optics')
    expect(one.actions).toStrictEqual(['createFolder optics'])
  })

  it('answers nothing and says why where the vault refused it', async () => {
    const one = createWindow({ folder: { error: NOTE_ERROR_CODE.ALREADY_EXISTS } })

    const made = await one.carry(invoke('app:folder.create', folder('physics')))

    expect(made).toBe('')
    expect(one.notices).toStrictEqual([words.alreadyExists])
  })
})
