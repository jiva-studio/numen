/**
 * What a document in a test can measure.
 *
 * The document a test runs in performs no layout: every element is a zero-size
 * box at the origin, and a range returns no measurements. The editor measures
 * its rendered text on its own frame after the test that mounted it has
 * finished, and a range with nothing to return throws where no handler catches
 * it.
 *
 * A range here behaves like the elements around it: it exists and has zero
 * size.
 */

const createEmptyRect = (): DOMRect => new DOMRect(0, 0, 0, 0)

/** An empty list of boxes, in the shape the DOM hands one over in. */
const noBoxes = (): DOMRectList => {
  const list: DOMRect[] = []
  const item = (pos: number): DOMRect | null => list[pos] ?? null
  const rects: DOMRectList = Object.assign(list, { item })
  return rects
}

Range.prototype.getBoundingClientRect = createEmptyRect
Range.prototype.getClientRects = noBoxes
