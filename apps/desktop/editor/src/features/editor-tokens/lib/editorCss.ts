/** What the editor tokens are written from: the three editor settings as the window holds them. */
interface EditorCssValues {
  readonly fontSize: number
  readonly lineHeight: number
  readonly fontFamily: string
}

/** The style rule that sets the editor tokens of the page. */
export function getEditorCss(_values: EditorCssValues): string {
  return ''
}
