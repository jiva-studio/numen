/**
 * What a fenced block can be written in.
 *
 * The word after the fence is looked up here; a word that is not here leaves
 * the block plain.
 */
import { LanguageDescription, LanguageSupport, StreamLanguage } from '@codemirror/language'
import { javascript } from '@codemirror/lang-javascript'
import { python } from '@codemirror/lang-python'
import { json } from '@codemirror/lang-json'
import { html } from '@codemirror/lang-html'
import { css } from '@codemirror/lang-css'
import { go } from '@codemirror/lang-go'
import { rust } from '@codemirror/lang-rust'
import { sql } from '@codemirror/lang-sql'
import { yaml } from '@codemirror/lang-yaml'
import { xml } from '@codemirror/lang-xml'
import { shell } from '@codemirror/legacy-modes/mode/shell'
import { toml } from '@codemirror/legacy-modes/mode/toml'
import { diff } from '@codemirror/legacy-modes/mode/diff'

const streamed = (mode: Parameters<typeof StreamLanguage.define>[0]) =>
  new LanguageSupport(StreamLanguage.define(mode))

export const LANGUAGES: readonly LanguageDescription[] = [
  LanguageDescription.of({
    name: 'javascript',
    alias: ['js', 'jsx', 'mjs', 'cjs', 'node'],
    load: async () => javascript({ jsx: true }),
  }),
  LanguageDescription.of({
    name: 'typescript',
    alias: ['ts', 'tsx'],
    load: async () => javascript({ jsx: true, typescript: true }),
  }),
  LanguageDescription.of({ name: 'python', alias: ['py'], load: async () => python() }),
  LanguageDescription.of({ name: 'json', load: async () => json() }),
  LanguageDescription.of({ name: 'html', alias: ['htm', 'vue'], load: async () => html() }),
  LanguageDescription.of({ name: 'css', load: async () => css() }),
  LanguageDescription.of({ name: 'go', alias: ['golang'], load: async () => go() }),
  LanguageDescription.of({ name: 'rust', alias: ['rs'], load: async () => rust() }),
  LanguageDescription.of({ name: 'sql', load: async () => sql() }),
  LanguageDescription.of({ name: 'yaml', alias: ['yml'], load: async () => yaml() }),
  LanguageDescription.of({ name: 'xml', alias: ['svg'], load: async () => xml() }),
  LanguageDescription.of({
    name: 'shell',
    alias: ['sh', 'bash', 'zsh', 'console'],
    load: async () => streamed(shell),
  }),
  LanguageDescription.of({ name: 'toml', load: async () => streamed(toml) }),
  LanguageDescription.of({ name: 'diff', load: async () => streamed(diff) }),
]

/**
 * The language one whole document is written in, by the name a fence would use.
 * A name no language here answers to leaves the document plain.
 */
export const wholly = async (name: string): Promise<LanguageSupport | null> => {
  const found = LanguageDescription.matchLanguageName(LANGUAGES, name, true)
  return found ? found.load() : null
}
