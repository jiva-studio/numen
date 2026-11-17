/** The size the plex was designed at, in pixels of the type a title is set in, and the lengths that go with it. */
export const DESIGNED_TYPE = 13
const ROW_HEIGHT = 22
const INDENT_STEP = 12
const MARGIN = 16
const PAD_SHARE = 0.25

/** The lengths of the list, in pixels, for the type a title is set in. */
export interface Measures {
  readonly rowHeight: number
  readonly indentStep: number
  readonly margin: number
  readonly pad: number
}

/** Every length is multiplied by what the type was multiplied by, so a row goes on holding its words. */
export function getMeasures(type: number): Measures {
  const delta = type > 0 ? type / DESIGNED_TYPE : 1
  const rowHeight = ROW_HEIGHT * delta
  return {
    rowHeight,
    indentStep: INDENT_STEP * delta,
    margin: MARGIN * delta,
    pad: Math.round(PAD_SHARE * rowHeight),
  }
}
