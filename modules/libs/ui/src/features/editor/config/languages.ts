/** Supported code block syntax highlighting languages for CodeMirror markdown. */
import { LanguageDescription, LanguageSupport, StreamLanguage } from '@codemirror/language'

type StreamParser = Parameters<typeof StreamLanguage.define>[0]

const createStreamLanguage = (mode: StreamParser) =>
  new LanguageSupport(StreamLanguage.define(mode))

export const LANGUAGES: readonly LanguageDescription[] = [
  LanguageDescription.of({
    name: 'javascript',
    alias: ['js', 'jsx', 'mjs', 'cjs', 'node'],
    load: () => import('@codemirror/lang-javascript').then((m) => m.javascript({ jsx: true })),
  }),
  LanguageDescription.of({
    name: 'typescript',
    alias: ['ts', 'tsx'],
    load: () =>
      import('@codemirror/lang-javascript').then((m) =>
        m.javascript({ jsx: true, typescript: true }),
      ),
  }),
  LanguageDescription.of({
    name: 'python',
    alias: ['py'],
    load: () => import('@codemirror/lang-python').then((m) => m.python()),
  }),
  LanguageDescription.of({
    name: 'json',
    load: () => import('@codemirror/lang-json').then((m) => m.json()),
  }),
  LanguageDescription.of({
    name: 'html',
    alias: ['htm', 'vue'],
    load: () => import('@codemirror/lang-html').then((m) => m.html()),
  }),
  LanguageDescription.of({
    name: 'css',
    load: () => import('@codemirror/lang-css').then((m) => m.css()),
  }),
  LanguageDescription.of({
    name: 'go',
    alias: ['golang'],
    load: () => import('@codemirror/lang-go').then((m) => m.go()),
  }),
  LanguageDescription.of({
    name: 'rust',
    alias: ['rs'],
    load: () => import('@codemirror/lang-rust').then((m) => m.rust()),
  }),
  LanguageDescription.of({
    name: 'sql',
    load: () => import('@codemirror/lang-sql').then((m) => m.sql()),
  }),
  LanguageDescription.of({
    name: 'yaml',
    alias: ['yml'],
    load: () => import('@codemirror/lang-yaml').then((m) => m.yaml()),
  }),
  LanguageDescription.of({
    name: 'xml',
    alias: ['svg'],
    load: () => import('@codemirror/lang-xml').then((m) => m.xml()),
  }),
  LanguageDescription.of({
    name: 'shell',
    alias: ['sh', 'bash', 'zsh', 'console'],
    load: () =>
      import('@codemirror/legacy-modes/mode/shell').then((m) => createStreamLanguage(m.shell)),
  }),
  LanguageDescription.of({
    name: 'toml',
    load: () =>
      import('@codemirror/legacy-modes/mode/toml').then((m) => createStreamLanguage(m.toml)),
  }),
  LanguageDescription.of({
    name: 'diff',
    load: () =>
      import('@codemirror/legacy-modes/mode/diff').then((m) => createStreamLanguage(m.diff)),
  }),
]

/**
 * The language one whole document is written in, by the name a fence would use.
 * A name no language here answers to leaves the document plain.
 */
export const loadLanguage = async (name: string): Promise<LanguageSupport | null> => {
  const found = LanguageDescription.matchLanguageName(LANGUAGES, name, true)
  return found ? found.load() : null
}
