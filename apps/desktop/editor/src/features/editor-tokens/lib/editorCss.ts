/** What the editor tokens are written from: the editor settings as the window holds them. */
interface EditorCssValues {
  readonly highlightActiveLine?: boolean
  readonly fontSize: number
  readonly lineHeight: number
  readonly fontFamily: string
}

/** The typeface each choice of the font family sets. The face of the theme sets nothing. */
const FAMILIES: ReadonlyMap<string, string> = new Map([
  ['sans', 'var(--numen-font-sans)'],
  ['serif', 'ui-serif, Georgia, "Times New Roman", serif'],
  ['mono', 'var(--numen-font-mono)'],
])

/** The style rule that sets the editor tokens of the page. */
export function getEditorCss(values: EditorCssValues): string {
  const name = values.fontFamily.trim()
  const family =
    FAMILIES.get(name) ?? (/^[\p{L}\p{M}\p{N} ._'()-]+$/u.test(name) ? `"${name}"` : '')
  const tokens = [
    `--numen-editor-font-size: ${values.fontSize}px`,
    `--numen-editor-line-height: ${values.lineHeight}`,
    ...(values.highlightActiveLine
      ? ['--numen-editor-active-line: var(--numen-selection-match)']
      : []),
    ...(family ? [`--numen-editor-font-family: ${family}`] : []),
  ]
  return `:root { ${tokens.join('; ')}; }`
}
