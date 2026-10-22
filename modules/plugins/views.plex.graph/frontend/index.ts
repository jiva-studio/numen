import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import PlexGraphView from './PlexGraphView.vue'
import { createPlexKind } from './kind'
import { usePlexView } from './model/usePlexView'

export { PlexGraphView, PlexGraphView as PlexTab }
export { createPlexKind, PLEX } from './kind'
export { createPlexTab, usePlexTab } from './model/usePlexTab'
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
}

export default definePlugin({
  activate,
})
