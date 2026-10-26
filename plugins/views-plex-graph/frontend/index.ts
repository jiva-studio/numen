import { Waypoints } from '@lucide/vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import PlexGraphView from './PlexGraphView.vue'
import { createPlexKind, PLEX } from './kind'
import { usePlexView } from './model/usePlexView'
import { WORDS } from './words'

export { PlexGraphView, PlexGraphView as PlexTab }
export { createPlexKind, PLEX } from './kind'
export { usePlexTab } from './model/usePlexTab'
export { usePlexView } from './model/usePlexView'
export { ITEMS, NONE, OFFERED, NEW_NOTE } from './lib/menu'
export * from './types'
export * from './words'

export function activate(context: PluginContext): void {
  const { workspace, notes } = context
  if (!workspace || !notes) {
    throw new Error('a plex needs the window it is drawn in and the notes of the vault')
  }
  const plexes = createPlexKind({ workspace, notes }, () =>
    usePlexView({ neighbourhood: (path) => notes.getNeighbourhood(path) }),
  )
  context.registerTabKind(plexes.kind)
  context.onVaultChange((event) => plexes.refresh(event))

  /** The note put in front in the plex the person was last in, and in a new plex where none is open. */
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
    scope: 'note',
    shortcut: 'mod+shift+p',
    icon: Waypoints,
    isNoteDestination: true,
    handler: (note) => travel(note.path),
  })

  context.registerContextMenu({
    id: 'travel',
    text: WORDS.travel,
    group: 'open',
    target: 'note',
    icon: Waypoints,
    handler: (target) => travel(target.path),
  })
  context.registerContextMenu({
    id: 'child',
    text: WORDS.child,
    parentId: 'relation',
    target: 'note',
    handler: (target) => {
      void notes.createInRelation(target.path, 'child')
    },
  })
  context.registerContextMenu({
    id: 'parent',
    text: WORDS.parent,
    parentId: 'relation',
    target: 'note',
    handler: (target) => {
      void notes.createInRelation(target.path, 'parent')
    },
  })
  context.registerContextMenu({
    id: 'jump',
    text: WORDS.jump,
    parentId: 'relation',
    target: 'note',
    handler: (target) => {
      void notes.createInRelation(target.path, 'jump')
    },
  })
}

export default definePlugin({
  activate,
})
