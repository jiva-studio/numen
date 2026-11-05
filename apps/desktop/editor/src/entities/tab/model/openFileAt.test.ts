/**
 * A file that is no note, opened at ranges of its text: its reader is given the ranges to light.
 */
import { describe, expect, it } from 'vitest'
import type { FileKind } from '@/shared/model/fileTypes'
import type { TextRange } from '@/shared/textRange'
import { fileOpeners, type FileOpenerDeps } from './openers'

/** A vault that answers with the kinds it is given. */
const createVault = (kinds: Record<string, FileKind>): FileOpenerDeps => ({
  fileKinds: async (paths) => {
    const found = new Map<string, FileKind>()
    for (const path of paths) {
      const one = kinds[path]
      if (one) found.set(path, one)
    }
    return found
  },
})

const OTHER: FileKind = { isNote: false, type: 'note' }

describe('a file opened at ranges of its text', () => {
  it('passes the ranges to the reader of the file in their order', async () => {
    const tabOpeners = fileOpeners(createVault({ 'library/Birds.bin': OTHER }))
    const readerCalls: unknown[][] = []
    tabOpeners.registerExtensionReader('bin', (path, selector, ranges) => {
      readerCalls.push([path, selector, ranges])
    })
    const ranges: readonly TextRange[] = [
      { start: 1500, end: 1580 },
      { start: 400, end: 1100 },
    ]

    await tabOpeners.openFileAt('library/Birds.bin', ranges)

    expect(readerCalls).toStrictEqual([['library/Birds.bin', null, ranges]])
  })
})
