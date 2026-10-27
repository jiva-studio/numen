/**
 * The files tab kind: one tree of the vault to a window, what it opens on, and
 * what a change to the vault does to it.
 */
import { describe, expect, it, vi } from 'vitest'
import { effectScope, nextTick, ref } from 'vue'
import { createFilesKind, FILES } from './kind'
import { ROOT } from './model/useFileTree'
import { createHost, type HostOptions } from './fixtures/host'
import type { Entry, FilesTabState, FileTree } from './types'

const createTreeOf = () => {
  const selectedPaths = ref<readonly string[]>([])
  const openFolder = vi.fn(async (_folder: string) => {})
  const revealPath = vi.fn(async (_path: string) => {})
  const refreshChanged = vi.fn(async () => {})
  const close = vi.fn()
  const entries = new Map<string, Entry>()
  const tree = {
    selectedPaths,
    openFolder,
    revealPath,
    refreshChanged,
    close,
    getEntryAt: (path: string) => entries.get(path) ?? null,
  } as unknown as FileTree
  return { tree, selectedPaths, openFolder, revealPath, refreshChanged, close, entries }
}

const createKindOver = (tree: FileTree, options: HostOptions = {}) => {
  const fixture = createHost(options)
  return { ...fixture, ...createFilesKind(fixture.host, () => tree) }
}

/** A tab of the kind, opened in a scope of its own the way the window opens one. */
const openTab = (kind: ReturnType<typeof createKindOver>['kind'], targetPath = '', id = FILES) => {
  const scope = effectScope(true)
  const state = scope.run(() => kind.open!(targetPath, id))!
  return { state, scope }
}

describe('a files tab', () => {
  it('is called Files and is one tab per window', () => {
    const { kind } = createKindOver(createTreeOf().tree)
    const { state } = openTab(kind)

    expect(kind.kind).toBe(FILES)
    expect(kind.getTitle?.(state)).toBe('Files')
    expect(kind.identity?.('anything')).toBe(FILES)
  })

  it('opens on the root of the vault, and reveals nothing where it was told nothing', () => {
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

  it('stops following the selection once the tab is gone', async () => {
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
    const { kind, refresh } = createFilesKind(fixture.host, () => trees.shift()!)
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
    const revealPath = vi.fn(async (_path: string) => {})
    const { kind } = createKindOver(createTreeOf().tree)
    const state = { list: { revealPath } } as unknown as FilesTabState

    await kind.navigate?.(state, 'Books/Guide.pdf', '')

    expect(revealPath).toHaveBeenCalledWith('Books/Guide.pdf')
  })
})
