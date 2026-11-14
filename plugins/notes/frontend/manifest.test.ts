/**
 * The place the manifest gives the editor's command that creates a note in the title bar.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { readMarkdownEditorPreferences } from './model/preferences'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  contributes: {
    commands?: readonly unknown[]
    slots?: readonly {
      slot: string
      command?: string
      view?: string
      group?: string
      order?: number
      when?: string
    }[]
  }
}

describe('the manifest', () => {
  it('places creating a note, by the full id of the editor command, in the group new of the title bar while a vault is open', () => {
    expect(manifest.contributes.slots).toStrictEqual([
      {
        slot: 'titleBar.commands',
        command: 'app:note.create',
        group: 'new',
        order: 10,
        when: 'window.isVaultOpen',
      },
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

interface ManifestSetting {
  key: string
  type: string
  title: string
  description: string
  default: unknown
  min?: number
  max?: number
  step?: number
  options?: { value: string; title: string }[]
}

const settings = (
  JSON.parse(readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8')) as {
    settings: ManifestSetting[]
  }
).settings
const byKey = (key: string) => settings.find((one) => one.key === key)

describe('the editor settings of the manifest', () => {
  const keys = ['tabSize', 'fontFamily', 'lineHeight', 'spellCheck', 'highlightActiveLine']

  it('declares the five editor settings with their defaults', () => {
    const defaults = readMarkdownEditorPreferences() as unknown as Record<string, unknown>
    for (const key of keys) {
      expect(byKey(key), key).toBeDefined()
      expect(byKey(key)?.default, key).toStrictEqual(defaults[key])
    }
  })

  it('declares tab size and line height as ranges with their bounds', () => {
    expect(byKey('tabSize')).toMatchObject({ type: 'range', min: 1, max: 8, step: 1 })
    expect(byKey('lineHeight')).toMatchObject({ type: 'range', min: 1.2, max: 2.4, step: 0.05 })
  })

  it('declares spell check and the active line as switches', () => {
    expect(byKey('spellCheck')?.type).toBe('boolean')
    expect(byKey('highlightActiveLine')?.type).toBe('boolean')
  })

  it('declares the font as a text with the interface font and the generic families as options', () => {
    const font = byKey('fontFamily')
    expect(font?.type).toBe('string')
    expect(font?.options?.map((one) => one.value)).toStrictEqual([
      '',
      'serif',
      'sans-serif',
      'monospace',
    ])
  })

  it('describes every setting in at most 64 characters', () => {
    for (const one of settings) {
      expect(one.description.length, one.key).toBeLessThanOrEqual(64)
    }
  })

  it('keeps the settings that were there', () => {
    expect(byKey('softWrap')?.default).toBe(true)
    expect(byKey('lineNumbers')?.default).toBe(false)
  })
})

describe('the readme settings list', () => {
  it('names every setting of the manifest', () => {
    const readme = readFileSync(resolve(import.meta.dirname, '../README.md'), 'utf8')
    for (const one of settings) {
      expect(readme, one.title).toContain(one.title)
    }
  })
})
