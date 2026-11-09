import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import PlexGraphView from './PlexGraphView.vue'
import { createPlexKind, PLEX } from './kind'
import { usePlexView } from './model/usePlexView'
import { readNesting } from './model/nesting'

export { PlexGraphView, PlexGraphView as PlexTab }
export { createPlexKind, PLEX } from './kind'
export { createPlexTab } from './model/usePlexTab'
export { usePlexView } from './model/usePlexView'
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
    {
      workspace,
      ...(context.commands ? { commands: context.commands } : {}),
      ...(context.slots ? { slots: context.slots } : {}),
      notes,
      relations,
      nesting: readNesting(context),
    },
    () => usePlexView({ neighbourhood: (path) => notes.getNeighbourhood(path) }),
  )
  context.registerTabKind(plexes.kind)
  context.onVaultChange((event) => plexes.refresh(event))

  context.registerCommand({
    id: 'graph.open',
    handler: async () => {
      await context.openTab?.(PLEX)
    },
  })
}

export default definePlugin({
  activate,
})
