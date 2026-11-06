/**
 * The place the manifest gives the editor's command that creates a note in the title bar.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  contributes: {
    commands?: readonly unknown[]
    slots?: readonly { slot: string; command?: string; view?: string; group?: string; order?: number }[]
  }
}

describe('the manifest', () => {
  it('places creating a note, by the full id of the editor command, in the group new of the title bar', () => {
    expect(manifest.contributes.slots).toStrictEqual([
      { slot: 'titleBar.commands', command: 'app:note.create', group: 'new', order: 10 },
    ])
  })

  it('declares no command of its own', () => {
    expect(Object.keys(manifest.contributes)).not.toContain('commands')
  })

  it('places nothing in the menu of a search result', () => {
    const entries = (manifest.contributes.slots ?? []).filter(
      (one) => one.slot === 'palette.searchResult',
    )

    expect(entries).toStrictEqual([])
  })
})

describe('the readme', () => {
  it('names the title bar button that creates a note', () => {
    const readme = readFileSync(resolve(import.meta.dirname, '../README.md'), 'utf8')

    expect(readme).toMatch(/title bar/i)
  })
})
