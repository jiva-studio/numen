/**
 * Tests for the files tab registration.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { FILES, type WindowHandle } from '@/entities/tab'
import { createFilesKind } from './kind'
import { ROOT } from './model/useFileTree'
import type { FilesTabDeps, FilesTabState, FileTree } from './types'

const stubTab = (id: string) => {
  const refreshChanged = vi.fn(async () => {})
  return { id, state: { list: { refreshChanged } } as unknown as FilesTabState, refreshChanged }
}

describe('a change to files', () => {
  it('reaches every files tab, the ones out of sight too', async () => {
    const front = stubTab('files:front')
    const hidden = stubTab('files:hidden')
    const handle = { each: () => [hidden, front] } as unknown as WindowHandle
    const files = createFilesKind(handle, () => ({}) as FileTree, {} as FilesTabDeps)

    await files.refreshChangedPaths(['a.md'], [])

    expect(front.refreshChanged).toHaveBeenCalledWith(['a.md'], [])
    expect(hidden.refreshChanged).toHaveBeenCalledWith(['a.md'], [])
  })

  it('is taken without a files tab open', async () => {
    const handle = { each: () => [] } as unknown as WindowHandle
    const files = createFilesKind(handle, () => ({}) as FileTree, {} as FilesTabDeps)

    await expect(files.refreshChangedPaths(['a.md'])).resolves.toBeUndefined()
  })
})

const createTreeOf = () => {
  const selectedPaths = ref<readonly string[]>([])
  const openFolder = vi.fn(async (_folder: string) => {})
  const close = vi.fn()
  const tree = { selectedPaths, openFolder, close } as unknown as FileTree
  return { tree, selectedPaths, openFolder, close }
}

const createKindOver = (tree: FileTree, deps: Partial<FilesTabDeps> = {}) =>
  createFilesKind({} as WindowHandle, () => tree, deps as FilesTabDeps).kind

describe('a files tab', () => {
  it('is called Files and is one tab per window', () => {
    const kind = createKindOver(createTreeOf().tree)
    const state = kind.open('', 'tab')

    expect(kind.kind).toBe(FILES)
    expect(kind.getTitle?.(state)).toBe('Files')
    expect(kind.identity?.('anything')).toBe(FILES)
  })

  it('opens on the root of the vault', () => {
    const { tree, openFolder } = createTreeOf()

    createKindOver(tree).open('', 'tab')

    expect(openFolder).toHaveBeenCalledExactlyOnceWith(ROOT)
  })

  it('lets go of its tree when it closes, and goes', () => {
    const { tree, close } = createTreeOf()
    const kind = createKindOver(tree)
    const state = kind.open('', 'tab')

    expect(kind.onClose?.(state, FILES)).toBe(true)
    expect(close).toHaveBeenCalledTimes(1)
  })
})

describe('a path selected in a files tab', () => {
  it('is reported with the tab it was selected in, the first of them', async () => {
    const { tree, selectedPaths } = createTreeOf()
    const onSelectPath = vi.fn<NonNullable<FilesTabDeps['onSelectPath']>>()
    const state = createKindOver(tree, { onSelectPath }).open('', 'tab')

    selectedPaths.value = ['notes/a.md', 'notes/b.md']
    await nextTick()

    expect(onSelectPath).toHaveBeenCalledExactlyOnceWith('notes/a.md', state)
  })

  it('is not reported where the selection is emptied', async () => {
    const { tree, selectedPaths } = createTreeOf()
    const onSelectPath = vi.fn<NonNullable<FilesTabDeps['onSelectPath']>>()
    createKindOver(tree, { onSelectPath }).open('', 'tab')

    selectedPaths.value = ['notes/a.md']
    await nextTick()
    onSelectPath.mockClear()
    selectedPaths.value = []
    await nextTick()

    expect(onSelectPath).not.toHaveBeenCalled()
  })

  it('is taken where the window asks for no report', async () => {
    const { tree, selectedPaths } = createTreeOf()
    createKindOver(tree).open('', 'tab')

    selectedPaths.value = ['notes/a.md']

    await expect(nextTick()).resolves.toBeUndefined()
  })
})

describe('a change to files without renames', () => {
  it('reaches a files tab with none', async () => {
    const tab = stubTab('files:front')
    const handle = { each: () => [tab] } as unknown as WindowHandle
    const files = createFilesKind(handle, () => ({}) as FileTree, {} as FilesTabDeps)

    await files.refreshChangedPaths(['a.md'])

    expect(tab.refreshChanged).toHaveBeenCalledExactlyOnceWith(['a.md'], [])
  })
})

describe('a path revealed in the files tab', () => {
  it('is revealed in the files tab opened for it', async () => {
    const revealPath = vi.fn(async (_path: string) => {})
    const openTab = vi.fn(async (_kind: string) => 'files:front')
    const getTabState = vi.fn(() => ({ list: { revealPath } }))
    const handle = { openTab, getTabState } as unknown as WindowHandle
    const files = createFilesKind(handle, () => ({}) as FileTree, {} as FilesTabDeps)

    await files.revealPath('notes/a.md')

    expect(openTab).toHaveBeenCalledExactlyOnceWith(FILES)
    expect(getTabState).toHaveBeenCalledExactlyOnceWith(FILES, 'files:front')
    expect(revealPath).toHaveBeenCalledExactlyOnceWith('notes/a.md')
  })

  it('is taken where the tab is gone', async () => {
    const handle = {
      openTab: async () => 'files:front',
      getTabState: () => undefined,
    } as unknown as WindowHandle
    const files = createFilesKind(handle, () => ({}) as FileTree, {} as FilesTabDeps)

    await expect(files.revealPath('notes/a.md')).resolves.toBeUndefined()
  })
})

describe('a files tab a linked tab moves', () => {
  it('reveals the path it is moved to, a note or not', async () => {
    const revealPath = vi.fn(async (_path: string) => {})
    const kind = createKindOver(createTreeOf().tree)
    const state = { list: { revealPath } } as unknown as FilesTabState

    await kind.navigate?.(state, 'Books/Guide.pdf', '')

    expect(revealPath).toHaveBeenCalledWith('Books/Guide.pdf')
  })

  it('is a tool the trailing pane opens with, after the agent', () => {
    const kind = createKindOver(createTreeOf().tree)

    expect(kind.isTool).toBe(true)
    expect(kind.placement).toStrictEqual({ pane: 'aside', order: 1 })
  })
})
