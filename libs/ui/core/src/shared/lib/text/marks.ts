/**
 * Pieces of a text that the caller draws as links.
 *
 * The caller names each piece as it is written; markdown-it turns every place
 * it stands, outside code, into the link every other mark becomes. What a piece
 * means is the caller's.
 */
import type { MarkdownIt, StateInline } from 'markdown-it'

/** A piece of a text drawn as a link. */
export interface LinkPiece {
  /** The piece as the text holds it. */
  readonly raw: string
  readonly href: string
  /** The text the link reads as. */
  readonly text: string
}

/** The pieces of one parse, by the code of their first character, longest first. */
export type LinkPieces = ReadonlyMap<number, readonly LinkPiece[]>

/** What a parse carries in its env. */
export type LinkPiecesEnv = { readonly pieces?: LinkPieces }

export const indexLinkPieces = (pieces: readonly LinkPiece[]): LinkPieces => {
  const index = new Map<number, LinkPiece[]>()
  for (const piece of pieces) {
    if (piece.raw === '') continue
    const code = piece.raw.charCodeAt(0)
    index.set(code, [...(index.get(code) ?? []), piece])
  }
  for (const list of index.values()) list.sort((a, b) => b.raw.length - a.raw.length)
  return index
}

const getPieces = (state: StateInline): LinkPieces | undefined =>
  (state.env as LinkPiecesEnv | undefined)?.pieces

const linkPiece = (state: StateInline, silent: boolean): boolean => {
  // A link inside a link's own text is not drawn.
  if (state.linkLevel > 0) return false
  const { src, pos, posMax } = state
  const found = getPieces(state)
    ?.get(src.charCodeAt(pos))
    ?.find((one) => pos + one.raw.length <= posMax && src.startsWith(one.raw, pos))
  if (!found) return false
  // A piece whose href does not pass is the text it is written as.
  if (!state.md.validateLink(found.href)) return false

  if (!silent) {
    state.push('link_open', 'a', 1).attrSet('href', found.href)
    state.push('text', '', 0).content = found.text
    state.push('link_close', 'a', -1)
  }
  state.pos = pos + found.raw.length
  return true
}

/** The characters markdown-it's own text rule stops at. */
const TERMINATORS = new Set([
  0x0a, 0x21, 0x23, 0x24, 0x25, 0x26, 0x2a, 0x2b, 0x2d, 0x3a, 0x3c, 0x3d, 0x3e, 0x40, 0x5b, 0x5c,
  0x5d, 0x5e, 0x5f, 0x60, 0x7b, 0x7d, 0x7e,
])

/** markdown-it's text rule, which also stops where a piece may begin. */
const text = (state: StateInline, silent: boolean): boolean => {
  const pieces = getPieces(state)
  let pos = state.pos
  while (pos < state.posMax) {
    const code = state.src.charCodeAt(pos)
    if (TERMINATORS.has(code) || pieces?.has(code)) break
    pos++
  }
  if (pos === state.pos) return false
  if (!silent) state.pending += state.src.slice(state.pos, pos)
  state.pos = pos
  return true
}

/** Draws the pieces a parse's env names as links. */
export const linkPieces = (md: MarkdownIt): void => {
  const parseLinkLabel = md.helpers.parseLinkLabel
  md.helpers.parseLinkLabel = (state, start, disableNested) => {
    if (disableNested) state.linkLevel++
    try {
      return parseLinkLabel(state, start, disableNested)
    } finally {
      if (disableNested) state.linkLevel--
    }
  }
  md.inline.ruler.at('text', text)
  md.inline.ruler.before('text', 'link_piece', linkPiece)
}
