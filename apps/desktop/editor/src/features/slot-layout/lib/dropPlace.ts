/** Where a dragged bar entry lands in a slot, from the boxes of the entries drawn there. */

/** The horizontal extent of one entry drawn in a slot, by its layout key. */
export interface EntryBox {
  readonly key: string
  readonly left: number
  readonly right: number
}

/** The key the entry goes before, none for the end of the slot, and the x of the insertion marker. */
export interface DropPlace {
  readonly before: string | undefined
  readonly x: number
}

/**
 * The place of a drop at x: before the first entry whose horizontal midpoint is right of x, or at
 * the end of the slot. An empty slot puts the marker at its own left edge.
 */
export function findDropPlace(boxes: readonly EntryBox[], x: number, slotLeft: number): DropPlace {
  const next = boxes.find((box) => (box.left + box.right) / 2 > x)
  if (next) return { before: next.key, x: next.left }
  const last = boxes.at(-1)
  return { before: undefined, x: last ? last.right : slotLeft }
}
