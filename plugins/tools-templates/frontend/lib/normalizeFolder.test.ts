import { describe, expect, it } from 'vitest'
import { normalizeFolder } from './normalizeFolder'

interface FolderCase {
  readonly raw: string
  readonly folder: string
}

const table = import.meta.glob('../../testdata/folder.json', {
  eager: true,
  import: 'default',
}) as Record<string, Record<string, readonly FolderCase[]>>

const cases = table['../../testdata/folder.json'] ?? {}

describe('the folder setting', () => {
  for (const name of [
    'reads an empty folder as Templates',
    'drops slashes and dot segments',
    'reads a parent segment as Templates',
  ]) {
    it(name, () => {
      const rows = cases[name] ?? []
      expect(rows.length).toBeGreaterThan(0)
      for (const row of rows) {
        expect(normalizeFolder(row.raw), JSON.stringify(row.raw)).toBe(row.folder)
      }
    })
  }
})
