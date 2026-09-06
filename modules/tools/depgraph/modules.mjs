/** Where the cruise finds its rules and its binary. The modules are the tools'. */
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

export { modules, root } from '../modules.mjs'

export const here = dirname(fileURLToPath(import.meta.url))
export const config = join(here, 'rules.cjs')
export const screens = join(here, 'screens.cjs')
export const depcruise = join(here, 'node_modules/.bin/depcruise')

/**
 * The modules whose folders are read as screens and shared folders, each with
 * one file under a screen the cruise has to have reached. A cruise that read
 * only a module's root would find no screen to judge and pass.
 *
 * `@numen/ui` is not here and will not be: it is a library of components, and
 * a component drawing another is what it is for. `@numen/mobile` is not here
 * either — the phone is one page with a note sheet on it, so which of its two
 * folders is a screen is a question nobody has answered.
 */
export const screened = new Map([
  ['@numen/editor', 'src/cards/deck.ts'],
  ['@numen/flashcards', 'src/decks/scheduling.ts'],
])

/**
 * The edges that break a rule today, by module, as the check prints them. The
 * list only shrinks: an entry naming an edge nobody draws any more is a rule
 * kept alive by a line nobody reads, and the check refuses that too.
 */
export const owed = new Map([
  [
    '@numen/editor',
    [
      // A deck and a stencil are edited as a note is, and a deck is scheduled
      // by a preset. The coupling is the domain's, not the folders': what the
      // cards screen takes is the note's editing and its tab state, and the
      // preset's core. Design rather than debt — worth recording so that a
      // fourth screen appearing here is read as a change and not as more of
      // the same.
      'no-screen-reaches-a-screen: src/cards/deck.ts → src/note/editing.ts',
      'no-screen-reaches-a-screen: src/cards/deck.ts → src/note/tab.ts',
      'no-screen-reaches-a-screen: src/cards/deck.ts → src/preset/core.ts',
      'no-screen-reaches-a-screen: src/cards/deck.test.ts → src/preset/core.ts',
      'no-screen-reaches-a-screen: src/cards/DeckTab.test.ts → src/preset/core.ts',
      'no-screen-reaches-a-screen: src/cards/stencil.ts → src/note/editing.ts',
      'no-screen-reaches-a-screen: src/cards/stencil.ts → src/note/tab.ts',
    ],
  ],
])

/** One box per folder of ours and one per package, which is the level a drawing reads at. */
export const COLLAPSE = '^(node_modules/@[a-z0-9-]+|node_modules|src)/[^/]+'
