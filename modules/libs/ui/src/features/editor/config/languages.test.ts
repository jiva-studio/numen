/**
 * The languages a fence can name.
 *
 * Names and aliases resolve to one description each, every description loads a
 * LanguageSupport on first use, and a grammar that fails to load fails only its
 * own language. Each test reads a freshly evaluated module.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LanguageDescription } from '@codemirror/language'
import type { LanguageSupport } from '@codemirror/language'
import type * as Languages from './languages'

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

const loadFresh = async (): Promise<typeof Languages> => {
  vi.resetModules()
  return import('./languages')
}

const find = (languages: typeof Languages, name: string, fuzzy = false) =>
  LanguageDescription.matchLanguageName(languages.LANGUAGES, name, fuzzy)

const getNodeNames = (support: LanguageSupport, code: string): Set<string> => {
  const names = new Set<string>()
  support.language.parser.parse(code).iterate({
    enter: (node) => {
      names.add(node.name)
    },
  })
  return names
}

afterEach(() => {
  vi.doUnmock('@codemirror/lang-python')
  vi.resetModules()
})

describe('LANGUAGES', () => {
  it('lists exactly the supported languages', async () => {
    const languages = await loadFresh()
    expect(languages.LANGUAGES.map((one) => one.name)).toEqual(Object.keys(ALIASES))
  })

  it('resolves every language by its name and each alias', async () => {
    const languages = await loadFresh()
    for (const [name, aliases] of Object.entries(ALIASES)) {
      for (const word of [name, ...aliases]) {
        expect(find(languages, word)?.name).toBe(name)
      }
    }
  })

  it('resolves names regardless of case', async () => {
    const languages = await loadFresh()
    expect(find(languages, 'Python')?.name).toBe('python')
    expect(find(languages, 'BASH')?.name).toBe('shell')
  })

  it('finds nothing for an unknown name', async () => {
    const languages = await loadFresh()
    expect(find(languages, 'brainfuck')).toBeNull()
    expect(find(languages, '')).toBeNull()
  })

  it('loads every language as a LanguageSupport of its own', async () => {
    const languages = await loadFresh()
    const grammars = new Set<unknown>()
    for (const name of Object.keys(ALIASES)) {
      const support = await find(languages, name)!.load()
      expect(support.language).toBeDefined()
      expect(support.extension).toBeDefined()
      grammars.add(support.language)
    }
    expect(grammars.size).toBe(Object.keys(ALIASES).length)
  })

  it('loads javascript with JSX', async () => {
    const languages = await loadFresh()
    const js = await languages.loadLanguage('javascript')
    expect(getNodeNames(js!, 'const a = <i>x</i>').has('JSXElement')).toBe(true)
  })

  it('loads typescript with JSX and with types', async () => {
    const languages = await loadFresh()
    const ts = await languages.loadLanguage('typescript')
    const names = getNodeNames(ts!, 'const a: number = 1; const b = <i>x</i>')
    expect(names.has('TypeAnnotation')).toBe(true)
    expect(names.has('JSXElement')).toBe(true)
  })

  it('answers a second load with the same instance', async () => {
    const languages = await loadFresh()
    const description = find(languages, 'python')!
    const first = await description.load()
    const second = await description.load()
    expect(second).toBe(first)
  })
})

describe('loadLanguage', () => {
  it('loads a language by alias', async () => {
    const languages = await loadFresh()
    const byAlias = await languages.loadLanguage('golang')
    expect(byAlias).toBe(await find(languages, 'go')!.load())
  })

  it('loads a language by a word that contains its name', async () => {
    const languages = await loadFresh()
    const fuzzy = await languages.loadLanguage('python3')
    expect(fuzzy).toBe(await find(languages, 'python')!.load())
  })

  it('answers none for an unknown name', async () => {
    const languages = await loadFresh()
    expect(await languages.loadLanguage('brainfuck')).toBeNull()
  })

  it('rejects when a grammar fails to import and leaves the other languages loading', async () => {
    vi.resetModules()
    vi.doMock('@codemirror/lang-python', () => {
      throw new Error('chunk failed')
    })
    const reloaded = await import('./languages')
    await expect(reloaded.loadLanguage('python')).rejects.toThrow()
    expect((await reloaded.loadLanguage('rust'))?.language).toBeDefined()
  })
})
