/**
 * The languages a fence can name.
 *
 * Names and aliases resolve to one description each, every description loads a
 * LanguageSupport on first use, and a grammar that fails to load fails only its
 * own language.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LanguageDescription, LanguageSupport } from '@codemirror/language'
import { LANGUAGES, loadLanguage } from './languages'

const ALIASES: Readonly<Record<string, readonly string[]>> = {
  javascript: ['js', 'jsx', 'mjs', 'cjs', 'node'],
  typescript: ['ts', 'tsx'],
  python: ['py'],
  json: [],
  html: ['htm', 'vue'],
  css: [],
  go: ['golang'],
  rust: ['rs'],
  sql: [],
  yaml: ['yml'],
  xml: ['svg'],
  shell: ['sh', 'bash', 'zsh', 'console'],
  toml: [],
  diff: [],
}

const find = (name: string) => LanguageDescription.matchLanguageName(LANGUAGES, name, true)

afterEach(() => {
  vi.doUnmock('@codemirror/lang-python')
  vi.resetModules()
})

describe('LANGUAGES', () => {
  it('lists exactly the supported languages', () => {
    expect(LANGUAGES.map((one) => one.name)).toEqual(Object.keys(ALIASES))
  })

  it.each(Object.entries(ALIASES))('resolves %s by name and every alias', (name, aliases) => {
    for (const word of [name, ...aliases]) {
      expect(find(word)?.name).toBe(name)
    }
  })

  it('resolves names regardless of case', () => {
    expect(find('Python')?.name).toBe('python')
    expect(find('BASH')?.name).toBe('shell')
  })

  it('finds nothing for an unknown name', () => {
    expect(find('brainfuck')).toBeNull()
    expect(find('')).toBeNull()
  })

  it.each(Object.keys(ALIASES))('loads %s as a LanguageSupport', async (name) => {
    const support = await find(name)!.load()
    expect(support).toBeInstanceOf(LanguageSupport)
  })

  it('loads typescript with a different grammar than javascript', async () => {
    const js = await find('javascript')!.load()
    const ts = await find('typescript')!.load()
    expect(ts.language).not.toBe(js.language)
  })

  it('answers a second load with the same instance', async () => {
    const description = find('python')!
    const first = await description.load()
    const second = await description.load()
    expect(second).toBe(first)
  })
})

describe('loadLanguage', () => {
  it('loads a language by alias', async () => {
    expect(await loadLanguage('golang')).toBeInstanceOf(LanguageSupport)
  })

  it('answers none for an unknown name', async () => {
    expect(await loadLanguage('brainfuck')).toBeNull()
  })

  it('rejects when a grammar fails to import and leaves the other languages loading', async () => {
    vi.resetModules()
    vi.doMock('@codemirror/lang-python', () => {
      throw new Error('chunk failed')
    })
    const fresh = await import('./languages')
    await expect(fresh.loadLanguage('python')).rejects.toThrow()
    expect(await fresh.loadLanguage('rust')).toBeInstanceOf(LanguageSupport)
  })
})
