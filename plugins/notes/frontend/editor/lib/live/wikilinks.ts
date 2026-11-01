/**
 * How a wikilink is drawn: `[[target]]`, `[[target|alias]]` and `[[target#heading]]` are one link.
 *
 * The whole link carries one class. While a selection touches it, or when marks are not concealed,
 * the brackets and the target of an alias are shown in the mark colour inside it; otherwise only
 * the shown text remains.
 */
import { syntaxTree } from '@codemirror/language'
import type { EditorState, Range } from '@codemirror/state'
import { Decoration, type DecorationSet } from '@codemirror/view'
import { isCode } from '../extensions/address'
import { getWikilinks, type Wikilink } from '../extensions/wikilink'
import { HIDDEN, mark } from './styles'

/** The wikilinks written between `from` and `to` outside code, at their positions in the document. */
export const findDrawnWikilinks = (state: EditorState, from: number, to: number): Wikilink[] => {
  const found: Wikilink[] = []
  const first = state.doc.lineAt(from).number
  const last = state.doc.lineAt(to).number
  for (let number = first; number <= last; number++) {
    const row = state.doc.line(number)
    if (!row.text.includes('[[')) continue
    for (const one of getWikilinks(row.text)) {
      const start = row.from + one.at
      if (isCode(syntaxTree(state).resolveInner(start, 1))) continue
      found.push({
        at: start,
        to: row.from + one.to,
        address: one.address,
        shownFrom: row.from + one.shownFrom,
        shownTo: row.from + one.shownTo,
      })
    }
  }
  return found
}

/** Whether a selection touches the link, its brackets included. */
export const isLinkSelected = (state: EditorState, link: Wikilink): boolean =>
  state.selection.ranges.some((range) => range.from <= link.to && range.to >= link.at)

/** What one wikilink is drawn with. */
export const drawWikilink = (link: Wikilink, isShownAsWritten: boolean): Range<Decoration>[] => {
  if (!isShownAsWritten) {
    return [
      HIDDEN.range(link.at, link.shownFrom),
      mark('cm-wikilink').range(link.shownFrom, link.shownTo),
      HIDDEN.range(link.shownTo, link.to),
    ]
  }
  const shownWithMarks = [
    mark('cm-wikilink').range(link.at, link.to),
    mark('cm-wikilink-mark').range(link.at, link.at + 2),
    mark('cm-wikilink-mark').range(link.to - 2, link.to),
  ]
  if (link.shownFrom > link.at + 2) {
    shownWithMarks.push(mark('cm-wikilink-mark').range(link.at + 2, link.shownFrom))
  }
  return shownWithMarks
}

/** Whether a node of the markdown syntax lies within the link, and so is part of it. */
export const isWithin = (links: readonly Wikilink[], from: number, to: number): boolean =>
  links.some((link) => from >= link.at && to <= link.to)

/** The marks of every wikilink between `from` and `to`; `isConcealing` hides the marks away from a selection. */
export const getWikilinkMarks = (
  state: EditorState,
  from: number,
  to: number,
  isConcealing: boolean,
): DecorationSet => {
  const ranges: Range<Decoration>[] = []
  for (const link of findDrawnWikilinks(state, from, to)) {
    ranges.push(...drawWikilink(link, !isConcealing || isLinkSelected(state, link)))
  }
  return Decoration.set(ranges, true)
}

/** Whether a position lies on a bracket of a wikilink. */
const isWikilinkBracket = (state: EditorState, pos: number): boolean => {
  const row = state.doc.lineAt(pos)
  if (!row.text.includes('[[')) return false
  return findDrawnWikilinks(state, row.from, row.from).some(
    (link) => pos >= link.at && pos < link.to,
  )
}

const MATCHED = Decoration.mark({ class: 'cm-matchingBracket' })
const UNMATCHED = Decoration.mark({ class: 'cm-nonmatchingBracket' })

/**
 * The highlight of a bracket and the one it pairs with. The brackets of a wikilink are one mark of
 * the link, so a bracket that belongs to one is not highlighted alone.
 */
export const renderBracketMatch = (
  match: {
    matched: boolean
    start: { from: number; to: number }
    end?: { from: number; to: number } | undefined
  },
  state: EditorState,
): Range<Decoration>[] => {
  const ends = match.end ? [match.start, match.end] : [match.start]
  if (ends.some((one) => isWikilinkBracket(state, one.from))) return []
  const decoration = match.matched ? MATCHED : UNMATCHED
  return ends.map((one) => decoration.range(one.from, one.to))
}
