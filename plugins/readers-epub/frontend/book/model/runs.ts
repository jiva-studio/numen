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
import { doc } from '@numen/ui'
import { countBytes, getUnitOffset } from '../lib/bytes'
import type { TextRange } from '@numen/ui'
import type { Mark } from '../lib/spread'

/** Single contiguous text run in rendered book. */
export interface Run {
  readonly at: number
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

const getPlaceInRun = (
  run: HTMLElement,
  into: number,
): { node: Text; offset: number } | undefined => {
  const walk = doc.createTreeWalker(run, NodeFilter.SHOW_TEXT)
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

const clipSpan = (runs: readonly Run[], span: TextRange): TextRange | undefined => {
  const first = runs[0]
  const last = runs.at(-1)
  if (!first || !last) return undefined
  if (span.end <= span.start) return span
  const textEnd = last.at + countBytes(last.element.textContent ?? '')
  if (span.end <= first.at || span.start >= textEnd) return undefined
  return { start: Math.max(span.start, first.at), end: Math.min(span.end, textEnd) }
}

const createRangeOver = (runs: readonly Run[], span: TextRange): Range | undefined => {
  const clipped = clipSpan(runs, span)
  if (!clipped) return undefined
  const { start, end } = clipped
  const startRun = findRun(runs, start)
  const endRun = findRun(runs, end)
  if (!startRun || !endRun) return undefined
  const endBytes = end > start ? end - endRun.at : countBytes(endRun.element.textContent ?? '')
  const startPoint = getPlaceInRun(startRun.element, start - startRun.at)
  const endPoint = getPlaceInRun(endRun.element, endBytes)
  if (!startPoint || !endPoint) return undefined
  const range = doc.createRange()
  range.setStart(startPoint.node, startPoint.offset)
  range.setEnd(endPoint.node, endPoint.offset)
  return range
}

/** The stretches that fall in the drawn document, as ranges over its nodes. */
export function createRangesOver(runs: readonly Run[], spans: readonly TextRange[]): Range[] {
  const renderedRanges: Range[] = []
  for (const span of spans) {
    const range = createRangeOver(runs, span)
    if (range) renderedRanges.push(range)
  }
  return renderedRanges
}
