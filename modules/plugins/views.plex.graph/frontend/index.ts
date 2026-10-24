import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import PlexGraphView from './PlexGraphView.vue'
import { createPlexKind } from './kind'
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

  context.registerContextMenu({
    id: 'travel',
    text: WORDS.travel,
    group: 'open',
    target: 'note',
    handler: (target) => {
      workspace.runCommand('travel', { path: target.path, title: target.name ?? '' })
    },
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
