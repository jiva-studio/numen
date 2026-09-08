/**
 * The wikilink as markdown-it reads it.
 *
 * `[[name]]` becomes the link every other mark becomes, carrying the address
 * it points at, so whatever handles a link handles this one.
 */
import type { MarkdownIt, StateInline } from 'markdown-it'
import { wikilinksIn } from './address'

const wikilink = (state: StateInline, silent: boolean): boolean => {
  const { src, pos } = state
  if (src.charCodeAt(pos) !== 0x5b || src.charCodeAt(pos + 1) !== 0x5b) return false
  const end = src.indexOf(']]', pos + 2)
  if (end < 0) return false

  const found = wikilinksIn(src.slice(pos, end + 2))[0]
  if (!found || found.at !== 0) return false
  // The address is held to what every other link is held to. Brackets around
  // an address that does not pass are the text they are written as.
  if (!state.md.validateLink(found.address)) return false

  if (!silent) {
    state.push('link_open', 'a', 1).attrSet('href', found.address)
    state.push('text', '', 0).content = found.text
    state.push('link_close', 'a', -1)
  }
  state.pos = pos + found.to
  return true
}

/** Reads `[[name]]` as a link. */
export const wikilinks = (md: MarkdownIt): void => {
  md.inline.ruler.before('link', 'wikilink', wikilink)
}
