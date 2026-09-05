/**
 * A name renamed on its way across the border between two of our own packages.
 *
 * A barrel is where a name is finally settled, so an application writing
 * `{ X as Y }` against one of ours is saying that our word for the thing and
 * its own word collide. That is the same collision a `View` suffix is, seen
 * from the other side of the boundary, and it is fixed by renaming one of the
 * two declarations rather than by papering over it at the call site.
 *
 * An alias against a package we did not write is ordinary disambiguation, and
 * so is one against the generated schema: those names are not ours to choose.
 * An alias between two files of one package is not this either — nothing there
 * has crossed a border, and the declaration is at hand.
 */

/** The packages whose names we choose. */
const OURS = ['@numen/ui', '@numen/wire', '@numen/desktop-ui', '@numen/flashcards-ui']

/**
 * owed are the borders this installation still renames across, and the list
 * only shrinks. Each is a name the window and the library both want; the fix
 * is a rename in one of them, not an entry here.
 */
export const owed = [
  // The window's own palette screen wraps the library's palette.
  'modules/apps/desktop/ui/src/Palette.vue: Palette as PaletteView',
  // The window's deck tab wraps the library's deck.
  'modules/apps/desktop/ui/src/cards/DeckTab.vue: Deck as DeckView',
  // The window's files tab has a row of its own.
  'modules/apps/desktop/ui/src/files/FilesTab.vue: Row as TreeRow',
]

/** The `{ … }` clause of every import and export naming a package. */
const CLAUSES = /(?:^|[\n;])\s*(?:import|export)\s+(?:type\s+)?\{([^}]*)\}\s*from\s*['"]([^'"]+)['"]/g

/**
 * Every name one file renames on its way in from another of our packages, as
 * `X as Y`.
 *
 * `default as X` is not a rename: a default export carries no name to keep.
 */
export function renames(source) {
  const found = []
  for (const clause of source.matchAll(CLAUSES)) {
    if (!OURS.includes(clause[2])) continue
    for (const part of clause[1].split(',')) {
      const named = part.trim().match(/^(?:type\s+)?([A-Za-z0-9_$]+)\s+as\s+([A-Za-z0-9_$]+)$/)
      if (named && named[1] !== 'default') found.push(`${named[1]} as ${named[2]}`)
    }
  }
  return found
}
