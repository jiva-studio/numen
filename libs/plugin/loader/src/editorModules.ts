import type { HOST_MODULES } from '@numen/plugin-sdk/host-modules'

type EditorModuleName = Extract<keyof typeof HOST_MODULES, `@codemirror/${string}` | `@lezer/${string}`>

export const EDITOR_MODULES = {
  '@codemirror/autocomplete': () => import('@codemirror/autocomplete'),
  '@codemirror/commands': () => import('@codemirror/commands'),
  '@codemirror/lang-css': () => import('@codemirror/lang-css'),
  '@codemirror/lang-go': () => import('@codemirror/lang-go'),
  '@codemirror/lang-html': () => import('@codemirror/lang-html'),
  '@codemirror/lang-javascript': () => import('@codemirror/lang-javascript'),
  '@codemirror/lang-json': () => import('@codemirror/lang-json'),
  '@codemirror/lang-markdown': () => import('@codemirror/lang-markdown'),
  '@codemirror/lang-python': () => import('@codemirror/lang-python'),
  '@codemirror/lang-rust': () => import('@codemirror/lang-rust'),
  '@codemirror/lang-sql': () => import('@codemirror/lang-sql'),
  '@codemirror/lang-xml': () => import('@codemirror/lang-xml'),
  '@codemirror/lang-yaml': () => import('@codemirror/lang-yaml'),
  '@codemirror/language': () => import('@codemirror/language'),
  '@codemirror/lint': () => import('@codemirror/lint'),
  '@codemirror/state': () => import('@codemirror/state'),
  '@codemirror/view': () => import('@codemirror/view'),
  '@lezer/common': () => import('@lezer/common'),
  '@lezer/highlight': () => import('@lezer/highlight'),
  '@lezer/lr': () => import('@lezer/lr'),
  '@lezer/markdown': () => import('@lezer/markdown'),
} satisfies Record<EditorModuleName, () => Promise<unknown>>
