/** Where the cruise finds its rules and its binary. The modules are the tools'. */
import { existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

export { modules, root } from '../modules.mjs'

export const here = dirname(fileURLToPath(import.meta.url))
export const config = join(here, 'rules.cjs')
export const screens = join(here, 'screens.cjs')

/**
 * The cruiser's own command. The install hoists to the source root, so the
 * first `node_modules` up the tree that holds it is the one answering.
 */
function installed(path) {
  for (let at = here; ; at = dirname(at)) {
    const found = join(at, 'node_modules', path)
    if (existsSync(found)) return found
    if (dirname(at) === at) throw new Error(`nothing installed answers for ${path}`)
  }
}

export const depcruise = installed('.bin/depcruise')

/**
 * The modules whose folders are read as screens and shared folders, each with
 * one file under a screen the cruise has to have reached. A cruise that read
 * only a module's root would find no screen to judge and pass.
 */
export const screened = new Map([
  ['@numen/editor', 'src/features/cards/deck-tab/deck.ts'],
  ['@numen/flashcards', 'src/decks/presets.ts'],
])

/**
 * The modules the screen rule does not read, each with the reason. A module is
 * here or it is above, and `check.mjs` refuses one that is in neither: a module
 * quietly outside a check is the same fault as a screen quietly reaching a
 * screen, one level up.
 */
export const unscreened = {
  '@numen/ui': 'a library of components, where one drawing another is the whole point of it',
  '@numen/wire': 'one file, with no folders to divide',
  '@numen/mobile':
    'one screen — App.vue mounts PlexPage alone, and note/ is the sheet that page draws over itself',
}

/**
 * The edges that break a rule today, by module, as the check prints them. The
 * list only shrinks: an entry naming an edge nobody draws any more is a rule
 * kept alive by a line nobody reads, and the check refuses that too.
 */
export const baseline = new Map([
  [
    '@numen/editor',
    [
      // A deck and a stencil are edited as a note is, and a deck is scheduled
      // by a preset. The coupling is the domain's, not the folders': what the
      // cards screen takes is the note's editing and its tab state, and the
      // preset's core. These three edges are the design; a fourth screen
      // appearing here is a change.
      'no-screen-reaches-a-screen: src/features/cards/deck-tab/deckTabs.ts → src/features/note/notes.ts',
      'no-screen-reaches-a-screen: src/features/cards/deck-tab/deckTabs.ts → src/features/note/tab.ts',
      'no-screen-reaches-a-screen: src/features/cards/deck-tab/deckTabs.ts → src/features/preset/core.ts',
      'no-screen-reaches-a-screen: src/features/cards/deck-tab/scheduler.ts → src/features/preset/core.ts',
      'no-screen-reaches-a-screen: src/features/cards/deck-tab/deckTabs.test.ts → src/features/preset/core.ts',
      'no-screen-reaches-a-screen: src/features/cards/deck-tab/DeckTab.test.ts → src/features/preset/core.ts',
      'no-screen-reaches-a-screen: src/features/cards/stencil-tab/stencilTabs.ts → src/features/note/notes.ts',
      'no-screen-reaches-a-screen: src/features/cards/stencil-tab/stencilTabs.ts → src/features/note/tab.ts',
    ],
  ],
])

/**
 * One box per folder of ours and one per package, which is the level a drawing
 * reads at. The install hoists to the source root, so a package is named from
 * the module it is reached from and the climb up is part of the name.
 */
export const COLLAPSE = '^((\\.\\./)*node_modules/@[a-z0-9-]+|(\\.\\./)*node_modules|src)/[^/]+'
