/**
 * The modules a window serves to plugin frontends, each at its path on the window's origin.
 *
 * A plugin bundle imports these by path and carries no copy of them, so every plugin runs on the
 * window's own instance: one Vue, one `@numen/ui`, one SDK, one CodeMirror. A second copy of
 * `@codemirror/state` or `@lezer/common` fails the `instanceof` checks of the first.
 */
export const HOST_MODULES = {
  vue: '/sdk/vue.js',
  '@numen/plugin-sdk': '/sdk/plugin-sdk.js',
  '@numen/ui': '/sdk/ui.js',
  '@codemirror/autocomplete': '/sdk/codemirror/autocomplete.js',
  '@codemirror/commands': '/sdk/codemirror/commands.js',
  '@codemirror/lang-css': '/sdk/codemirror/lang-css.js',
  '@codemirror/lang-go': '/sdk/codemirror/lang-go.js',
  '@codemirror/lang-html': '/sdk/codemirror/lang-html.js',
  '@codemirror/lang-javascript': '/sdk/codemirror/lang-javascript.js',
  '@codemirror/lang-json': '/sdk/codemirror/lang-json.js',
  '@codemirror/lang-markdown': '/sdk/codemirror/lang-markdown.js',
  '@codemirror/lang-python': '/sdk/codemirror/lang-python.js',
  '@codemirror/lang-rust': '/sdk/codemirror/lang-rust.js',
  '@codemirror/lang-sql': '/sdk/codemirror/lang-sql.js',
  '@codemirror/lang-xml': '/sdk/codemirror/lang-xml.js',
  '@codemirror/lang-yaml': '/sdk/codemirror/lang-yaml.js',
  '@codemirror/language': '/sdk/codemirror/language.js',
  '@codemirror/lint': '/sdk/codemirror/lint.js',
  '@codemirror/state': '/sdk/codemirror/state.js',
  '@codemirror/view': '/sdk/codemirror/view.js',
  '@lezer/common': '/sdk/lezer/common.js',
  '@lezer/highlight': '/sdk/lezer/highlight.js',
  '@lezer/lr': '/sdk/lezer/lr.js',
  '@lezer/markdown': '/sdk/lezer/markdown.js',
} as const
