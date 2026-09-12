/** Where the cruise finds its rules and its binary. The modules are the tools'. */
import { existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

export { modules, root } from '../modules.mjs'

export const here = dirname(fileURLToPath(import.meta.url))
export const config = join(here, 'rules.cjs')
export const layers = join(here, 'layers.cjs')

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
 * The modules whose folders are read as layers, each with one file standing
 * under a layer the cruise has to have reached. A cruise that read only a
 * module's root would find no layer to judge and pass. What each layer may
 * reach is `layers.cjs`.
 */
export const layered = new Map([
  ['@numen/ui', 'src/features/cards/deck.ts'],
  ['@numen/editor', 'src/widgets/deck-editor/composables/useDeckTabs.ts'],
  ['@numen/flashcards', 'src/decks/presets.ts'],
])

/**
 * The modules no boundary rule reads, each with the reason. A module is here or
 * it is above, and `check.mjs` refuses one that is in neither: a module quietly
 * outside a check is the same fault as a screen quietly reaching a screen, one
 * level up.
 */
export const unlayered = {
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
      // A slice says what its tab draws, and the component says what state it
      // is drawn from, so the two segments name each other. The naming is the
      // ring: the factory belongs at the top of the slice, not inside one of
      // its segments, and it moves there when the segments take their
      // standard names.
      'no-folder-going-round: src/widgets/agent-chat/components → src/widgets/agent-chat/composables',
      'no-folder-going-round: src/widgets/agent-chat/composables → src/widgets/agent-chat/components',
      'no-folder-going-round: src/widgets/book-reader/components → src/widgets/book-reader/composables',
      'no-folder-going-round: src/widgets/book-reader/composables → src/widgets/book-reader/components',
      'no-folder-going-round: src/widgets/deck-editor/components → src/widgets/deck-editor/composables',
      'no-folder-going-round: src/widgets/deck-editor/composables → src/widgets/deck-editor/components',
      'no-folder-going-round: src/widgets/document-viewer/components → src/widgets/document-viewer/composables',
      'no-folder-going-round: src/widgets/document-viewer/composables → src/widgets/document-viewer/components',
      'no-folder-going-round: src/widgets/file-manager/components → src/widgets/file-manager/composables',
      'no-folder-going-round: src/widgets/file-manager/composables → src/widgets/file-manager/components',
      'no-folder-going-round: src/widgets/preset-editor/components → src/widgets/preset-editor/preset-settings',
      'no-folder-going-round: src/widgets/preset-editor/composables → src/widgets/preset-editor/components',
      'no-folder-going-round: src/widgets/preset-editor/preset-settings → src/widgets/preset-editor/composables',
      'no-folder-going-round: src/widgets/settings/components → src/widgets/settings/composables',
      'no-folder-going-round: src/widgets/settings/composables → src/widgets/settings/components',
      'no-folder-going-round: src/widgets/stencil-editor/components → src/widgets/stencil-editor/composables',
      'no-folder-going-round: src/widgets/stencil-editor/composables → src/widgets/stencil-editor/components',
      'no-folder-going-round: src/widgets/text-editor/components → src/widgets/text-editor/composables',
      'no-folder-going-round: src/widgets/text-editor/composables → src/widgets/text-editor/components',
      // The media entity is exercised against a real window of tabs, which is
      // the only thing that can say what opening one does. What media is built
      // on is `entities/tab/@x/media`; this is what its test mounts.
      'no-entities-slice-reaches-a-slice: src/entities/media/kind.test.ts → src/entities/tab/openers.ts',
      'no-entities-slice-reaches-a-slice: src/entities/media/kind.test.ts → src/entities/tab/windowTabs.ts',
      'no-entities-slice-reaches-a-slice: src/entities/media/kind.test.ts → src/entities/tab/workspace.ts',
      // A note left unsaved is flushed on the way out, and the test of the
      // store's quitting is where the two meet.
      'no-entities-reaches-above-itself: src/entities/note/notes.quitting.test.ts → src/features/file-conflict/flushing.ts',
    ],
  ],
  [
    '@numen/ui',
    [
      // An editor is drawn in a pane of the workspace, and a tab switched away
      // from and come back to measures its text again. That is the contract
      // between the two features, and the story is where it is held. The
      // editor itself reaches nothing of the workspace.
      'no-features-slice-reaches-a-slice: src/features/editor/Editor.stories.ts → src/features/workspace/node.ts',
      'no-features-slice-reaches-a-slice: src/features/editor/Editor.stories.ts → src/features/workspace/pane/index.ts',
      // The fixtures a plex is arranged from are arranged by the code they
      // stand for.
      'no-folder-going-round: src/features/plex/arrange → src/features/plex/fixtures',
      'no-folder-going-round: src/features/plex/fixtures → src/features/plex/arrange',
    ],
  ],
])

/**
 * One box per folder of ours and one per package, which is the level a drawing
 * reads at. The install hoists to the source root, so a package is named from
 * the module it is reached from and the climb up is part of the name.
 */
export const COLLAPSE = '^((\\.\\./)*node_modules/@[a-z0-9-]+|(\\.\\./)*node_modules|src)/[^/]+'
