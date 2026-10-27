/**
 * A stand-in for the core reading the wikilinks of a text: each `[[name]]` is `name://name`, and
 * it reaches the path `notes` gives it, or nothing.
 */
export const createLinkResolver =
  (notes: Readonly<Record<string, string>>) =>
  async (text: string): Promise<ReadonlyMap<string, string>> =>
    new Map(
      [...text.matchAll(/\[\[([^[\]|#]+)[^[\]]*\]\]/g)].map((match) => {
        const address = `name://${match[1]!.trim()}`
        return [address, notes[address] ?? '']
      }),
    )
