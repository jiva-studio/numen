/**
 * What a document in a test can measure.
 *
 * The document a test runs in lays nothing out: an element is a box of no size
 * at the origin, and a range answers nothing at all. The editor measures the
 * text it drew on a frame of its own, after the test that mounted it is over,
 * and a range with nothing to answer with throws where nobody is listening.
 *
 * A range here answers the way the elements around it do: it is there, and it
 * takes up no room.
 */

const nowhere = (): DOMRect => new DOMRect(0, 0, 0, 0)

/** An empty list of boxes, in the shape the DOM hands one over in. */
const noBoxes = (): DOMRectList => {
  const list: DOMRect[] = []
  const item = (at: number): DOMRect | null => list[at] ?? null
  return Object.assign(list, { item }) as unknown as DOMRectList
}

Range.prototype.getBoundingClientRect = nowhere
Range.prototype.getClientRects = noBoxes
