/**
 * The runs of a drawn document: where each of them begins in the book's text,
 * where it stands across the columns, and what part of the drawn nodes an
 * offset falls on.
 *
 * A run is an element carrying `data-offset`, the byte offset at which its text
 * begins in the book's text. The markup writes them in the order of those
 * offsets and they are kept in it, so the run an offset falls in is the last
 * one beginning at or before it.
 */
import { bytesIn, unitsIn } from './bytes'
import type { Mark, Span } from './spread'

/** One run of the book's text as it is drawn. */
export interface Run {
  /** Where the run begins, in bytes of the book's text. */
  readonly at: number
  /** The element the run is drawn in. */
  readonly element: HTMLElement
}

/** Every run of a drawn document, in the order the markup sets them. */
export function runsIn(paper: HTMLElement): Run[] {
  const found: Run[] = []
  for (const element of paper.querySelectorAll<HTMLElement>('[data-offset]')) {
    const said = Number(element.dataset['offset'])
    if (Number.isFinite(said)) found.push({ at: said, element })
  }
  return found
}

/**
 * Where each run stands, measured from the near edge of the columns. The first
 * of a run's rectangles is the one that counts: a run broken over a column edge
 * has one in each column, and it begins in the first.
 */
export function marksIn(runs: readonly Run[], origin: number): Mark[] {
  const placed: Mark[] = []
  for (const run of runs) {
    const first = run.element.getClientRects()[0]
    if (first) placed.push({ at: run.at, x: first.left - origin })
  }
  return placed
}

/** The run an offset falls in: the last one beginning at or before it. */
export function runAt(runs: readonly Run[], at: number): Run | undefined {
  let found: Run | undefined
  for (const run of runs) {
    if (run.at > at) break
    found = run
  }
  return found
}

/**
 * The offset a place named inside the document stands at: the run the named
 * element falls in, or the first run after a name standing between runs.
 */
export function offsetAt(
  paper: HTMLElement,
  runs: readonly Run[],
  fragment: string,
): number | undefined {
  if (fragment === '') return undefined

  let named: HTMLElement | undefined
  for (const element of paper.querySelectorAll<HTMLElement>('[id]')) {
    if (element.id === fragment) {
      named = element
      break
    }
  }
  if (!named) return undefined

  const run =
    named.closest<HTMLElement>('[data-offset]') ?? named.querySelector<HTMLElement>('[data-offset]')
  if (run) {
    const said = Number(run.dataset['offset'])
    return Number.isFinite(said) ? said : undefined
  }
  for (const after of runs) {
    if (named.compareDocumentPosition(after.element) & Node.DOCUMENT_POSITION_FOLLOWING) {
      return after.at
    }
  }
  return undefined
}

/**
 * Where an offset stands inside a run: the text node it falls in, and how far
 * into that node it reaches. The distance from the run's own offset is a number
 * of bytes, and only `unitsIn` turns one of those into a place in a string.
 */
const inside = (run: HTMLElement, into: number): { node: Text; offset: number } | undefined => {
  const walk = document.createTreeWalker(run, NodeFilter.SHOW_TEXT)
  let counted = 0
  let node = walk.nextNode() as Text | null
  while (node) {
    const bytes = bytesIn(node.data)
    if (counted + bytes >= into) return { node, offset: unitsIn(node.data, into - counted) }
    counted += bytes
    node = walk.nextNode() as Text | null
  }
  return undefined
}

/** One stretch of the book's text as a range over the nodes the markup carries. */
const rangeOver = (runs: readonly Run[], span: Span): Range | undefined => {
  const opens = runAt(runs, span.begins)
  const closes = runAt(runs, span.ends)
  if (!opens || !closes) return undefined
  const from = inside(opens.element, span.begins - opens.at)
  const to = inside(closes.element, span.ends - closes.at)
  if (!from || !to) return undefined
  const range = document.createRange()
  range.setStart(from.node, from.offset)
  range.setEnd(to.node, to.offset)
  return range
}

/** The stretches that fall in the drawn document, as ranges over its nodes. */
export function rangesOver(runs: readonly Run[], spans: readonly Span[]): Range[] {
  const drawn: Range[] = []
  for (const span of spans) {
    const range = rangeOver(runs, span)
    if (range) drawn.push(range)
  }
  return drawn
}
