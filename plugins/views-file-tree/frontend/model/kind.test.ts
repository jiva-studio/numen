/**
 * The files tab kind: one tree of the vault to a window, what it opens on, and
 * what a change to the vault does to it.
 */
import { describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import FilesTab from '../ui/FilesTab.vue'
import { createFilesKind, FILES } from './kind'
import { createFileTree, ROOT } from './useFileTree'
import { createHost, type HostOptions } from '../fixtures/host'
import type { Entry, FilesTabState, FileTree } from '../types'

const createTreeOf = () => {
  const selectedPaths = ref<readonly string[]>([])
  const openFolder = vi.fn(async (_folder: string) => undefined)
  const revealPath = vi.fn(async (_path: string) => undefined)
  const refreshChanged = vi.fn(async () => undefined)
  const close = vi.fn()
  const entries = new Map<string, Entry>()
  const tree = {
    state: { selectedPaths },
    folders: { openFolder },
    revealPath,
    sync: { refreshChanged },
    close,
    queries: { getEntryAt: (path: string) => entries.get(path) ?? null },
  } as unknown as FileTree
  return { tree, selectedPaths, openFolder, revealPath, refreshChanged, close, entries }
}

const createKindOver = (tree: FileTree, options: HostOptions = {}) => {
  const fixture = createHost(options)
  return { ...fixture, ...createFilesKind(fixture.host, () => tree, FilesTab) }
}

const openTab = (kind: ReturnType<typeof createKindOver>['kind'], targetPath = '', id = FILES) => {
  const scope = effectScope(true)
  const state = scope.run(() => kind.open!(targetPath, id))!
  return { state, scope }
}

describe('a files tab', () => {
  it('is called Files and allows opening multiple tabs', () => {
    const { kind } = createKindOver(createTreeOf().tree)
    const { state } = openTab(kind)

    expect(kind.kind).toBe(FILES)
    expect(kind.getTitle?.(state)).toBe('Files')
    expect(kind.identity).toBeUndefined()
  })

  it('opens on the root of the vault, and reveals nothing where it is given nothing', () => {
    const { tree, openFolder, revealPath } = createTreeOf()

    openTab(createKindOver(tree).kind)

    expect(openFolder).toHaveBeenCalledExactlyOnceWith(ROOT)
    expect(revealPath).not.toHaveBeenCalled()
  })

  it('reveals the path it was opened on', () => {
    const { tree, revealPath } = createTreeOf()

    openTab(createKindOver(tree).kind, 'notes/a.md')

    expect(revealPath).toHaveBeenCalledExactlyOnceWith('notes/a.md')
  })

  it('releases its tree when it closes, and goes', () => {
    const { tree, close } = createTreeOf()
    const { kind } = createKindOver(tree)
    const { state } = openTab(kind)

    expect(kind.onClose?.(state, FILES)).toBe(true)
    expect(close).toHaveBeenCalledTimes(1)
  })

  it('is a tool the trailing pane opens with, after the agent, under the tree icon', () => {
    const { kind } = createKindOver(createTreeOf().tree)

    expect(kind.isTool).toBe(true)
    expect(kind.placement).toStrictEqual({ pane: 'aside', order: 1 })
    expect(kind.icon).toBeDefined()
  })
})

describe('a path selected in a files tab', () => {
  const note = (path: string): Entry => ({
    path,
    name: 'a',
    isFolder: false,
    isNote: true,
    type: 'note',
  })

  it('moves the tabs linked to it, the first path of the selection', async () => {
    const { tree, selectedPaths, entries } = createTreeOf()
    entries.set('notes/a.md', note('notes/a.md'))
    const { kind, recorded } = createKindOver(tree)
    openTab(kind, '', 'files:one')

    selectedPaths.value = ['notes/a.md', 'notes/b.md']
    await nextTick()

    expect(recorded.synced).toStrictEqual([['files:one', 'notes/a.md', 'a']])
  })

  it('moves nothing for a folder, a file that is not a note, or an emptied selection', async () => {
    const { tree, selectedPaths, entries } = createTreeOf()
    entries.set('notes', { ...note('notes'), isFolder: true })
    entries.set('Cover.png', { ...note('Cover.png'), isNote: false })
    const { kind, recorded } = createKindOver(tree)
    openTab(kind)

    for (const paths of [['notes'], ['Cover.png'], []]) {
      selectedPaths.value = paths
      await nextTick()
    }

    expect(recorded.synced).toStrictEqual([])
  })

  it('stops following the selection once the tab is closed', async () => {
    const { tree, selectedPaths, entries } = createTreeOf()
    entries.set('notes/a.md', note('notes/a.md'))
    const { kind, recorded } = createKindOver(tree)
    const { scope } = openTab(kind)
    scope.stop()

    selectedPaths.value = ['notes/a.md']
    await nextTick()

    expect(recorded.synced).toStrictEqual([])
  })
})

describe('a change to the vault', () => {
  it('reaches every files tab with the paths and the renames it carries', async () => {
    const one = createTreeOf()
    const other = createTreeOf()
    const trees = [one.tree, other.tree]
    const fixture = createHost()
    const { kind, refresh } = createFilesKind(fixture.host, () => trees.shift()!, FilesTab)
    kind.open!('', 'files:one')
    kind.open!('', 'files:other')
    const renames = [{ from: 'a.md', to: 'b.md' }]

    await refresh({ paths: ['a.md'], renamed: renames })

    expect(one.refreshChanged).toHaveBeenCalledWith(['a.md'], renames)
    expect(other.refreshChanged).toHaveBeenCalledWith(['a.md'], renames)
  })

  it('reaches a tab with nothing where it is empty, and no tab once it has closed', async () => {
    const { tree, refreshChanged } = createTreeOf()
    const { kind, refresh } = createKindOver(tree)
    const { state } = openTab(kind)

    await refresh({})
    kind.onClose?.(state, FILES)
    await refresh({ paths: ['a.md'] })

    expect(refreshChanged).toHaveBeenCalledExactlyOnceWith([], [])
  })
})

describe('a files tab a linked tab moves', () => {
  it('reveals the path it is moved to, a note or not', async () => {
    const revealPath = vi.fn(async (_path: string) => undefined)
    const { kind } = createKindOver(createTreeOf().tree)
    const state = { list: { revealPath } } as unknown as FilesTabState

    await kind.navigate?.(state, 'Books/Guide.pdf', '')

    expect(revealPath).toHaveBeenCalledWith('Books/Guide.pdf')
  })
})

describe('a files tab and the open note', () => {
  const entry = (path: string, isFolder = false): Entry => ({
    path,
    name: path.split('/').at(-1)!,
    isFolder,
    isNote: !isFolder,
    type: isFolder ? 'folder' : 'note',
  })

  const folders: Record<string, readonly Entry[]> = {
    '': [entry('Projects', true), entry('Inbox.md')],
    Projects: [entry('Projects/Garden', true)],
    'Projects/Garden': [entry('Projects/Garden/Beds.md')],
  }

  const createRealKind = (isRevealOpenNote?: () => boolean) => {
    const fixture = createHost({ folders })
    const files = { list: fixture.host.files.list }
    const options = isRevealOpenNote ? { isRevealOpenNote } : undefined
    const created = (
      createFilesKind as unknown as (
        host: unknown,
        createTree: () => FileTree,
        component: unknown,
        options?: unknown,
      ) => ReturnType<typeof createFilesKind>
    )(fixture.host, () => createFileTree(files), FilesTab, options)
    return created
  }

  const openRealTab = (kind: ReturnType<typeof createFilesKind>['kind'], path = '') => {
    const scope = effectScope(true)
    return scope.run(() => kind.open!(path, FILES))! as FilesTabState
  }

  const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

  it('opens the folders above a note a linked tab moved to and selects it', async () => {
    const { kind } = createRealKind(() => true)
    const state = openRealTab(kind)
    await settle()

    await kind.navigate?.(state, 'Projects/Garden/Beds.md', 'Beds')

    expect(state.list.queries.isFolderOpen('Projects')).toBe(true)
    expect(state.list.queries.isFolderOpen('Projects/Garden')).toBe(true)
    expect(state.list.state.selectedPaths.value).toStrictEqual(['Projects/Garden/Beds.md'])
  })

  it('leaves the tree as it is when a linked tab moves and expanding is off', async () => {
    const { kind } = createRealKind(() => false)
    const state = openRealTab(kind)
    await settle()
    const before = [...state.list.queries.openRows.value]

    await kind.navigate?.(state, 'Projects/Garden/Beds.md', 'Beds')

    expect(state.list.queries.isFolderOpen('Projects')).toBe(false)
    expect(state.list.queries.isFolderOpen('Projects/Garden')).toBe(false)
    expect(state.list.queries.openRows.value).toStrictEqual(before)
    expect(state.list.state.selectedPaths.value).toStrictEqual([])
  })

  it('shows a note the person asked for although expanding is off', async () => {
    const { kind } = createRealKind(() => false)
    const state = openRealTab(kind)
    await settle()

    await kind.navigate?.(state, 'Projects/Garden/Beds.md', '')

    expect(state.list.queries.isFolderOpen('Projects/Garden')).toBe(true)
    expect(state.list.state.selectedPaths.value).toStrictEqual(['Projects/Garden/Beds.md'])
  })

  it('follows the setting as it changes, with no restart', async () => {
    let isOn = false
    const { kind } = createRealKind(() => isOn)
    const state = openRealTab(kind)
    await settle()

    await kind.navigate?.(state, 'Projects/Garden/Beds.md', 'Beds')
    expect(state.list.state.selectedPaths.value).toStrictEqual([])

    isOn = true
    await kind.navigate?.(state, 'Projects/Garden/Beds.md', 'Beds')
    expect(state.list.state.selectedPaths.value).toStrictEqual(['Projects/Garden/Beds.md'])

    isOn = false
    await kind.navigate?.(state, 'Inbox.md', 'Inbox')
    expect(state.list.state.selectedPaths.value).toStrictEqual(['Projects/Garden/Beds.md'])
  })

  it('reveals the path a tab opens on although expanding is off', async () => {
    const { kind } = createRealKind(() => false)
    const state = openRealTab(kind, 'Projects/Garden/Beds.md')
    await settle()

    expect(state.list.queries.isFolderOpen('Projects/Garden')).toBe(true)
    expect(state.list.state.selectedPaths.value).toStrictEqual(['Projects/Garden/Beds.md'])
  })
})
