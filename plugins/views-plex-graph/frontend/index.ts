import { Waypoints } from '@lucide/vue'
import {
  definePlugin,
  formatErrorMessage,
  getRelationErrorWords,
  NOTE_SCOPE,
  NOTE_TARGET,
  type ConnectionRole,
  type PluginContext,
} from '@numen/plugin-sdk'
import PlexGraphView from './PlexGraphView.vue'
import { createPlexKind, PLEX } from './kind'
import { usePlexView } from './model/usePlexView'
import { readNesting } from './model/nesting'
import { WORDS } from './words'

export { PlexGraphView, PlexGraphView as PlexTab }
export { createPlexKind, PLEX } from './kind'
export { usePlexTab } from './model/usePlexTab'
export { usePlexView } from './model/usePlexView'
export { ITEMS, NONE, OFFERED, NEW_NOTE } from './lib/menu'
export * from './types'
export * from './words'

export function activate(context: PluginContext): void {
  const { workspace, notes, relations } = context
  if (!workspace || !notes || !relations) {
    throw new Error(
      'a plex needs the window it is drawn in, the notes of the vault and their relationships',
    )
  }
  const plexes = createPlexKind(
    { workspace, notes, relations, nesting: readNesting(context) },
    () => usePlexView({ neighbourhood: (path) => notes.getNeighbourhood(path) }),
  )
  context.registerTabKind(plexes.kind)
  context.onVaultChange((event) => plexes.refresh(event))

  const reportFailure = (error: unknown): void => workspace.showError(formatErrorMessage(error))

  /** Makes an untitled note in a relation to another, and says why where the core made none. */
  const createRelated = async (from: string, role: ConnectionRole): Promise<void> => {
    const made = await relations.createRelated({ from, role })
    if (!made.ok) workspace.showError(getRelationErrorWords(made.error))
  }

  /** Opens the note in the plex that was last active, or in a new plex when none is open. */
  const travel = async (path: string): Promise<void> => {
    await context.navigateTab?.(PLEX, path)
  }

  context.registerCommand({
    id: 'plex',
    title: WORDS.newPlex,
    shortcut: 'mod+shift+x',
    icon: Waypoints,
    handler: async () => {
      await context.openTab?.(PLEX)
    },
  })
  context.registerCommand({
    id: 'travel',
    title: WORDS.travel,
    scope: NOTE_SCOPE,
    shortcut: 'mod+shift+p',
    icon: Waypoints,
    isNoteDestination: true,
    handler: (note) => travel(note.path),
  })

  context.registerContextMenu({
    id: 'travel',
    text: WORDS.travel,
    group: 'open',
    target: NOTE_TARGET,
    icon: Waypoints,
    handler: (target) => travel(target.path),
  })
  context.registerContextMenu({
    id: 'child',
    text: WORDS.child,
    parentId: 'relation',
    target: NOTE_TARGET,
    handler: (target) => {
      createRelated(target.path, 'child').catch(reportFailure)
    },
  })
  context.registerContextMenu({
    id: 'parent',
    text: WORDS.parent,
    parentId: 'relation',
    target: NOTE_TARGET,
    handler: (target) => {
      createRelated(target.path, 'parent').catch(reportFailure)
    },
  })
  context.registerContextMenu({
    id: 'jump',
    text: WORDS.jump,
    parentId: 'relation',
    target: NOTE_TARGET,
    handler: (target) => {
      createRelated(target.path, 'jump').catch(reportFailure)
    },
  })
}

export default definePlugin({
  activate,
})
