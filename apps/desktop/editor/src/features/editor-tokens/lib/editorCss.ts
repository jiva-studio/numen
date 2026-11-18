/** What the editor tokens are written from: the three editor settings as the window holds them. */
interface EditorCssValues {
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
  const family = FAMILIES.get(values.fontFamily)
  const tokens = [
    `--numen-editor-font-size: ${values.fontSize}px`,
    `--numen-editor-line-height: ${values.lineHeight}`,
    ...(family ? [`--numen-editor-font-family: ${family}`] : []),
  ]
  return `:root { ${tokens.join('; ')}; }`
}
