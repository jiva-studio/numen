import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import PlexGraphView from './ui/PlexGraphView.vue'
import { createPlexKind, PLEX } from './model/kind'
import { usePlexView } from './model/usePlexView'
import { readNesting } from './model/nesting'

export { PlexGraphView, PlexGraphView as PlexTab }
export { createPlexKind, PLEX } from './model/kind'
export { createPlexTab } from './model/usePlexTab'
export { usePlexView } from './model/usePlexView'
export * from './types'
export * from './words'

export function activate(context: PluginContext): void {
  const { workspace } = context.ui
  const { notes, relations } = context.vault
  if (!workspace || !notes || !relations) {
    throw new Error(
      'a plex needs the window it is drawn in, the notes of the vault and their relationships',
    )
  }
  const plexes = createPlexKind(
    {
      workspace,
      ...(context.ui.commands ? { commands: context.ui.commands } : {}),
      slots: context.ui.slots,
      notes,
      relations,
      nesting: readNesting(context),
    },
    () => usePlexView({ listRelatedNotes: (path) => notes.listRelatedNotes(path) }),
    PlexGraphView,
  )
  context.register.tabKind(plexes.kind)
  context.events.onVaultChange((event) => plexes.refresh(event))

  context.register.command({
    id: 'graph.open',
    handler: async (note) => {
      await context.tabs.navigate?.(PLEX, note.path)
    },
  })
}

export default definePlugin({
  activate,
})
