/**
 * What the files tab kind hands the file manager, and what it does with the
 * file manager's calls.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import { WORDS as words } from '@/shared/words'
import { cards, presets, WORDS as cardWords } from '@/entities/deck'
import { ANSWER_WORDS, invocationOf, openDestination, runInvocation } from '@/features/command-palette'
import { running } from '@/entities/artifact'
import { createFilesKind as buildFilesKind } from '@/pages/file-manager'
import type { FilesTabDeps } from '@/pages/file-manager/types'
import { createFilesKind } from './files'

vi.mock('@/entities/deck', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/entities/deck')>()),
  cards: { createDeck: vi.fn(), createStencil: vi.fn() },
  presets: { createPreset: vi.fn() },
}))
vi.mock('@/entities/artifact', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/entities/artifact')>()),
  running: { fetchArtifact: vi.fn() },
}))
vi.mock('@/features/command-palette', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/features/command-palette')>()),
  runInvocation: vi.fn(async () => undefined),
  openDestination: vi.fn(async () => undefined),
}))
vi.mock('@/pages/file-manager', () => ({
  createFilesKind: vi.fn(() => ({ kind: {} })),
  useFileTree: vi.fn(),
}))

type Held = Parameters<typeof createFilesKind>[0]['held']

/** A window whose files tabs are the ones given, in front or last as told. */
const holding = (
  tabs: { id: string; state: unknown }[],
  front: { kind: string | null; id: string } | null,
  last: { id: string } | null,
): Held =>
  ({
    handle: {
      each: (kind: string) => (kind === 'files' ? tabs : []),
      front: () => front,
      last: (kind: string) => (kind === 'files' ? last : null),
    },
  }) as unknown as Held

const noteEntry = { path: 'a.md', name: 'a.md', isFolder: false, kind: 'note' }

const setup = (over: Record<string, unknown> = {}) => {
  const syncPathFromTab = vi.fn(() => true)
  const writeMessage = vi.fn()
  const openNewFile = vi.fn()
  const createUrl = vi.fn()
  const getTarget = vi.fn(() => ({
    tab: 't',
    kind: null,
    vault: { id: 'v', name: 'V' },
    made: {},
    isReady: true,
    path: 'target',
    title: '',
    file: '',
    source: null,
  }))
  const commandDeps = vi.fn(() => ({ marker: 'deps' }))
  const runCommand = vi.fn()
  const createUntitled = vi.fn()
  const applyPathChanges = vi.fn()
  const canRun = vi.fn()
  const makes = shallowRef(new Map<string, unknown>())
  const dragged = shallowRef<readonly string[]>([])
  const destinations = { marker: 'destinations' }
  createFilesKind({
    core: { createUrl },
    tabOpeners: { openNewFile },
    held: holding([], { kind: 'files', id: 'front' }, { id: 'last' }),
    runs: { canRun },
    editing: { notes: { applyPathChanges }, making: { createUntitled } },
    vaults: { makes },
    getTarget,
    runCommand,
    commandDeps,
    dragged,
    writeMessage,
    destinations,
    tabSync: { syncPathFromTab },
    ...over,
  } as never)
  const deps = vi.mocked(buildFilesKind).mock.calls.at(-1)![2] as FilesTabDeps
  return {
    deps,
    syncPathFromTab,
    writeMessage,
    openNewFile,
    createUrl,
    getTarget,
    commandDeps,
    runCommand,
    createUntitled,
    applyPathChanges,
    canRun,
    makes,
    destinations,
    dragged,
  }
}

const stateOf = (entry: unknown) => ({ list: { getEntryAt: () => entry } }) as never

beforeEach(() => {
  vi.clearAllMocks()
})

describe('choosing a note in the files tab', () => {
  it('syncs the tab that holds the state, whichever tab is in front', () => {
    const mine = stateOf(noteEntry)
    const held = holding(
      [
        { id: 'one', state: stateOf(noteEntry) },
        { id: 'two', state: mine },
      ],
      { kind: 'files', id: 'front' },
      { id: 'last' },
    )
    const { deps, syncPathFromTab } = setup({ held })

    deps.onSelectPath!('a.md', mine)

    expect(syncPathFromTab).toHaveBeenCalledWith('two', 'a.md', 'a.md')
  })

  it('syncs the tab in front where no tab holds the state', () => {
    const held = holding([{ id: 'one', state: {} }], { kind: 'files', id: 'front' }, { id: 'last' })
    const { deps, syncPathFromTab } = setup({ held })

    deps.onSelectPath!('a.md', stateOf(noteEntry))

    expect(syncPathFromTab).toHaveBeenCalledWith('front', 'a.md', 'a.md')
  })

  it('syncs the files tab the person was last in where the front is another kind', () => {
    const held = holding([], { kind: 'note', id: 'front' }, { id: 'last' })
    const { deps, syncPathFromTab } = setup({ held })

    deps.onSelectPath!('a.md', stateOf(noteEntry))

    expect(syncPathFromTab).toHaveBeenCalledWith('last', 'a.md', 'a.md')
  })

  it('syncs the files tab the person was last in where nothing is in front', () => {
    const held = holding([], null, { id: 'last' })
    const { deps, syncPathFromTab } = setup({ held })

    deps.onSelectPath!('a.md', stateOf(noteEntry))

    expect(syncPathFromTab).toHaveBeenCalledWith('last', 'a.md', 'a.md')
  })

  it('syncs nothing where the window holds no files tab', () => {
    const held = holding([], null, null)
    const { deps, syncPathFromTab } = setup({ held })

    deps.onSelectPath!('a.md', stateOf(noteEntry))

    expect(syncPathFromTab).not.toHaveBeenCalled()
  })

  it('syncs a note by its kind even where its path does not look like one', () => {
    const entry = { path: 'a.txt', name: 'a.txt', isFolder: false, kind: 'note' }
    const { deps, syncPathFromTab } = setup()

    deps.onSelectPath!('a.txt', stateOf(entry))

    expect(syncPathFromTab).toHaveBeenCalledWith('front', 'a.txt', 'a.txt')
  })

  it('syncs a file by its path where its kind is not note but the path is one', () => {
    const entry = { path: 'a.md', name: 'a.md', isFolder: false, kind: 'file' }
    const { deps, syncPathFromTab } = setup()

    deps.onSelectPath!('a.md', stateOf(entry))

    expect(syncPathFromTab).toHaveBeenCalledWith('front', 'a.md', 'a.md')
  })

  it('syncs nothing for a file that is neither a note by kind nor by path', () => {
    const entry = { path: 'a.png', name: 'a.png', isFolder: false, kind: 'file' }
    const { deps, syncPathFromTab } = setup()

    deps.onSelectPath!('a.png', stateOf(entry))

    expect(syncPathFromTab).not.toHaveBeenCalled()
  })

  it('syncs nothing for a folder, whatever it is called', () => {
    const entry = { path: 'dir', name: 'dir', isFolder: true, kind: 'note' }
    const { deps, syncPathFromTab } = setup()

    deps.onSelectPath!('dir', stateOf(entry))

    expect(syncPathFromTab).not.toHaveBeenCalled()
  })

  it('syncs nothing for a path the list does not hold', () => {
    const { deps, syncPathFromTab } = setup()

    deps.onSelectPath!('a.md', stateOf(null))

    expect(syncPathFromTab).not.toHaveBeenCalled()
  })

  it('is no trouble for a window that links no tabs', () => {
    const { deps } = setup({ tabSync: undefined })

    expect(() => deps.onSelectPath!('a.md', stateOf(noteEntry))).not.toThrow()
  })
})

describe('opening a destination from the files tab', () => {
  const landing = { path: 'a.md', title: 'A' } as never

  it('is handled by the linked tab where one takes the note', () => {
    const { deps, syncPathFromTab } = setup()

    deps.openDestination(landing, undefined)

    expect(syncPathFromTab).toHaveBeenCalledWith('front', 'a.md', 'A')
    expect(openDestination).not.toHaveBeenCalled()
  })

  it('goes to the window where no linked tab takes the note', () => {
    const { deps, syncPathFromTab, destinations } = setup()
    syncPathFromTab.mockReturnValue(false)

    deps.openDestination(landing, undefined)

    expect(openDestination).toHaveBeenCalledWith(landing, destinations)
  })

  it('goes to the window where the window links no tabs', () => {
    const { deps, destinations } = setup({ tabSync: undefined })

    deps.openDestination(landing, undefined)

    expect(openDestination).toHaveBeenCalledWith(landing, destinations)
  })

  it('goes to the window for a destination that is not a note', () => {
    const png = { path: 'a.png', title: 'A' } as never
    const { deps, syncPathFromTab, destinations } = setup()

    deps.openDestination(png, undefined)

    expect(syncPathFromTab).not.toHaveBeenCalled()
    expect(openDestination).toHaveBeenCalledWith(png, destinations)
  })

  it('uses the state it is given to find the tab', () => {
    const mine = {}
    const held = holding([{ id: 'two', state: mine }], { kind: 'files', id: 'front' }, null)
    const { deps, syncPathFromTab } = setup({ held })

    deps.openDestination(landing, mine as never)

    expect(syncPathFromTab).toHaveBeenCalledWith('two', 'a.md', 'A')
  })

  it('finds the tab in front where it is given no state, even if a tab holds none', () => {
    const held = holding([{ id: 'bare', state: undefined }], { kind: 'files', id: 'front' }, null)
    const { deps, syncPathFromTab } = setup({ held })

    deps.openDestination(landing, undefined)

    expect(syncPathFromTab).toHaveBeenCalledWith('front', 'a.md', 'A')
  })
})

describe('what the files tab asks of the window', () => {
  it('runs a command over the first path, with the rest as the others and what the vault made', () => {
    const made = { thumbnail: 'ready' }
    const { deps, runCommand, makes } = setup()
    makes.value = new Map([['a.md', made]])

    deps.runCommand('delete', ['a.md', 'b.md', 'c.md'], 'A', 'tab' as never)

    expect(runCommand).toHaveBeenCalledWith('delete', {
      vault: { id: 'v', name: 'V' },
      tab: 't',
      isReady: true,
      kind: null,
      path: 'a.md',
      title: 'A',
      file: 'a.md',
      source: 'tab',
      made,
      others: ['b.md', 'c.md'],
    })
  })

  it('runs a command over a path the vault made nothing for with an empty record', () => {
    const { deps, runCommand } = setup()

    deps.runCommand('delete', ['a.md'], 'A', 'tab' as never)

    expect(runCommand.mock.calls[0]![1].made).toStrictEqual({})
  })

  it('moves a path through the move command, answered in the command words', async () => {
    const { deps, getTarget, commandDeps } = setup()
    const answered = vi.mocked(runInvocation).mockResolvedValueOnce(undefined)

    await deps.movePath('a.md', 'dir/a.md')

    expect(answered).toHaveBeenCalledWith(
      invocationOf('move', { ...getTarget(), path: 'a.md' }, 'dir/a.md'),
      commandDeps(),
      ANSWER_WORDS,
    )
  })

  it('creates a folder through the createFolder command, answered in the command words', async () => {
    const { deps, getTarget, commandDeps } = setup()

    await deps.createFolder('dir')

    expect(runInvocation).toHaveBeenCalledWith(
      invocationOf('createFolder', getTarget(), 'dir'),
      commandDeps(),
      ANSWER_WORDS,
    )
  })

  it('remembers the paths being dragged', () => {
    const { deps, dragged } = setup()

    deps.setDraggedPaths(['a.md'])

    expect(dragged.value).toEqual(['a.md'])
  })

  it('creates a note in a folder and answers where it stands', async () => {
    const { deps, createUntitled } = setup()
    createUntitled.mockResolvedValue({ path: 'dir/Untitled.md' })

    expect(await deps.createNote('dir')).toBe('dir/Untitled.md')
    expect(createUntitled).toHaveBeenCalledWith('dir', [])
  })

  it('answers no path where no note was made', async () => {
    const { deps, createUntitled } = setup()
    createUntitled.mockResolvedValue(null)

    expect(await deps.createNote('dir')).toBe('')
  })

  it('creates a deck under the name it is given', async () => {
    const { deps, openNewFile } = setup()
    vi.mocked(cards.createDeck).mockResolvedValue({ ok: true, value: { path: 'dir/D.md' } } as never)

    expect(await deps.createDeck('dir', 'D')).toBe('dir/D.md')

    expect(cards.createDeck).toHaveBeenCalledWith('D', 'dir')
    expect(openNewFile).not.toHaveBeenCalled()
  })

  it('creates a stencil under the name it is given, with a first field', async () => {
    const { deps } = setup()
    vi.mocked(cards.createStencil).mockResolvedValue({ ok: true, value: { path: 'dir/S.md' } } as never)

    expect(await deps.createStencil('dir', 'S')).toBe('dir/S.md')

    expect(cards.createStencil).toHaveBeenCalledWith('S', 'dir', [cardWords.newField])
  })

  it('creates a preset under the name it is given', async () => {
    const { deps } = setup()
    vi.mocked(presets.createPreset).mockResolvedValue({ ok: true, value: { path: 'dir/P.md' } } as never)

    expect(await deps.createPreset('dir', 'P')).toBe('dir/P.md')

    expect(presets.createPreset).toHaveBeenCalledWith('P', 'dir')
  })

  it('imports an address into a folder', async () => {
    const { deps, createUrl } = setup()
    createUrl.mockResolvedValue({ ok: true, value: { path: 'dir/U.md' } })

    expect(await deps.importUrl('dir', 'https://example.test')).toBe('dir/U.md')

    expect(createUrl).toHaveBeenCalledWith('https://example.test', 'dir')
  })

  it('shows an error as an error message', () => {
    const { deps, writeMessage } = setup()

    deps.showError('broken')

    expect(writeMessage).toHaveBeenCalledWith('broken', 'error')
  })

  it('answers whether a run can go from the window', () => {
    const { deps, canRun } = setup()
    canRun.mockReturnValue(true)
    const run = { id: 'r' } as never

    expect(deps.canRun!(run)).toBe(true)
    expect(canRun).toHaveBeenCalledWith(run)
  })

  it('answers the artifact states the vault made for a path, and none for another', () => {
    const made = { thumbnail: 'ready' }
    const { deps, makes } = setup()
    makes.value = new Map([['a.md', made]])

    expect(deps.getArtifactStates!('a.md')).toBe(made)
    expect(deps.getArtifactStates!('b.md')).toStrictEqual({})
  })
})

describe('an address imported from the files tab', () => {
  it('fetches what the note points at and tells the notes its path changed', async () => {
    const { deps, createUrl, applyPathChanges } = setup()
    createUrl.mockResolvedValue({ ok: true, value: { path: 'dir/U.md' } })

    await deps.importUrl('dir', 'https://example.test')

    await vi.waitFor(() => expect(applyPathChanges).toHaveBeenCalledWith(['dir/U.md']))
    expect(running.fetchArtifact).toHaveBeenCalledWith('dir/U.md')
  })

  it('says why the fetch failed and does not tell the notes', async () => {
    const { deps, createUrl, applyPathChanges, writeMessage } = setup()
    createUrl.mockResolvedValue({ ok: true, value: { path: 'dir/U.md' } })
    const failure = new Error('offline')
    vi.mocked(running.fetchArtifact).mockRejectedValueOnce(failure)

    await deps.importUrl('dir', 'https://example.test')

    await vi.waitFor(() => expect(writeMessage).toHaveBeenCalledWith(formatErrorMessage(failure), 'error'))
    expect(applyPathChanges).not.toHaveBeenCalled()
  })

  it('fetches nothing where the vault refused the address', async () => {
    const { deps, createUrl, applyPathChanges, writeMessage } = setup()
    createUrl.mockResolvedValue({ ok: false, error: 'occupied' })

    await deps.importUrl('dir', 'https://example.test')

    expect(writeMessage.mock.calls).toEqual([[words.errors.occupied, 'error']])
    expect(running.fetchArtifact).not.toHaveBeenCalled()
    expect(applyPathChanges).not.toHaveBeenCalled()
  })
})

describe('what the files tab asks, with nothing to go on', () => {
  it('syncs a destination with no title under an empty title', () => {
    const { deps, syncPathFromTab } = setup()

    deps.openDestination({ path: 'a.md' } as never, undefined)

    expect(syncPathFromTab).toHaveBeenCalledWith('front', 'a.md', '')
  })

  it('runs a command over no paths with an empty path', () => {
    const { deps, runCommand } = setup()

    deps.runCommand('delete', [], 'A', 'tab' as never)

    expect(runCommand.mock.calls[0]![1]).toMatchObject({ path: '', file: '', others: [] })
  })
})
