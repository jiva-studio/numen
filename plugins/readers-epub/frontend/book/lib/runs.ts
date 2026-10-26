/**
 * The runs of a drawn document: where each of them begins in the book's text,
 * at its current position across the columns, and what part of the drawn nodes an
 * offset falls on.
 *
 * A run is an element carrying `data-offset`, the byte offset at which its text
 * begins in the book's text. The markup writes them in the order of those
 * offsets and they are kept in it, so the run an offset falls in is the last
 * one beginning at or before it.
 */
import { countBytes, getUnitOffset } from './bytes'
import type { Span } from '@numen/ui'
import type { Mark } from './spread'

/** Single contiguous text run in rendered book. */
export interface Run {
  /** Byte offset where the text run begins. */
  readonly at: number
  /** DOM element containing the rendered text run. */
  readonly element: HTMLElement
}

/** Every run of a drawn document, in the order the markup sets them. */
export function getRuns(paper: HTMLElement): Run[] {
  const found: Run[] = []
  for (const element of paper.querySelectorAll<HTMLElement>('[data-offset]')) {
    const offset = Number(element.dataset['offset'])
    if (Number.isFinite(offset)) found.push({ at: offset, element })
  }
  return found
}

/**
 * of a run's rectangles is the one that counts: a run broken over a column edge
 * has one in each column, and it begins in the first.
 */
export function createMarks(runs: readonly Run[], origin: number): Mark[] {
  const placedMarks: Mark[] = []
  for (const run of runs) {
    const first = run.element.getClientRects()[0]
    if (first) placedMarks.push({ at: run.at, x: first.left - origin })
  }
  return placedMarks
}

/** The run an offset falls in: the last one beginning at or before it. */
export function findRun(runs: readonly Run[], offset: number): Run | undefined {
  let found: Run | undefined
  for (const run of runs) {
    if (run.at > offset) break
    found = run
  }
  return found
}

/**
 * element falls in, or the first run after a name standing between runs.
 */
export function findOffset(
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
    const offset = Number(run.dataset['offset'])
    return Number.isFinite(offset) ? offset : undefined
  }
  for (const after of runs) {
    if (named.compareDocumentPosition(after.element) & Node.DOCUMENT_POSITION_FOLLOWING) {
      return after.at
    }
  }
  return undefined
}

/**
 * into that node it reaches. The distance from the run's own offset is a number
 * of bytes, and only `getUnitOffset` turns one of those into a place in a string.
 */
const getPlaceInRun = (
  run: HTMLElement,
  into: number,
): { node: Text; offset: number } | undefined => {
  const walk = document.createTreeWalker(run, NodeFilter.SHOW_TEXT)
  let counted = 0
  let node = walk.nextNode() as Text | null
  while (node) {
    const bytes = countBytes(node.data)
    if (counted + bytes >= into) return { node, offset: getUnitOffset(node.data, into - counted) }
    counted += bytes
    node = walk.nextNode() as Text | null
  }
  return undefined
}

/** One stretch of the book's text as a range over the nodes the markup carries. */
const rangeOver = (runs: readonly Run[], span: Span): Range | undefined => {
  const opens = findRun(runs, span.from)
  const closes = findRun(runs, span.to)
  if (!opens || !closes) return undefined
  const from = getPlaceInRun(opens.element, span.from - opens.at)
  const to = getPlaceInRun(closes.element, span.to - closes.at)
  if (!from || !to) return undefined
  const range = document.createRange()
  range.setStart(from.node, from.offset)
  range.setEnd(to.node, to.offset)
  return range
}

/** The stretches that fall in the drawn document, as ranges over its nodes. */
export function rangesOver(runs: readonly Run[], spans: readonly Span[]): Range[] {
  const renderedRanges: Range[] = []
  for (const span of spans) {
    const range = rangeOver(runs, span)
    if (range) renderedRanges.push(range)
  }
  return renderedRanges
}
