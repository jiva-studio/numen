/**
 * Tests for the files tab registration.
 */
import { describe, expect, it, vi } from 'vitest'
import type { WindowHandle } from '@/entities/tab'
import { createFilesKind } from './kind'
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
