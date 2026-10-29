/**
 * A stand-in for the core extracting the wikilinks of a text: each
 * `[[name|alias]]` becomes `name://name`, reads as its alias or its name, and
 * resolves to the path `notes` gives it, or to nothing.
 */
import type { TextLink } from '@numen/plugin-sdk'

export const createLinkResolver =
  (notes: Readonly<Record<string, string>>) =>
  async (text: string): Promise<readonly TextLink[]> => {
    const found = new Map<string, TextLink>()
    for (const match of text.matchAll(/\[\[([^[\]|#]+)([^[\]|]*)(?:\|([^[\]]*))?\]\]/g)) {
      const address = `name://${match[1]!.trim()}`
      found.set(match[0], {
        raw: match[0],
        address,
        text: (match[3] ?? match[1]! + match[2]!).trim(),
        path: notes[address] ?? '',
      })
    }
    return [...found.values()]
  }
