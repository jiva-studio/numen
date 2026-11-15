import { computed } from 'vue'
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import PlexGraphView from './ui/PlexGraphView.vue'
import { createPlexKind, PLEX } from './model/kind'
import { usePlexView } from './model/usePlexView'
import { readNesting } from './model/nesting'
import { readNames } from './model/names'
import { readLook } from './model/look'

export { PlexGraphView, PlexGraphView as PlexTab }
export { createPlexKind, PLEX } from './model/kind'
export { createPlexTab } from './model/usePlexTab'
export { usePlexView } from './model/usePlexView'
export * from './types'
export * from './words'

export function activate(plugin: PluginContext, host: HostServices): void {
  const { workspace } = host.ui
  const { notes, relations } = host.vault
  if (!workspace || !notes || !relations) {
    throw new Error(
      'a plex needs the window it is drawn in, the notes of the vault and their relationships',
    )
  }
  const names = readNames(plugin)
  const depth = computed(() => (names.showNames.value ? 2 : 1))
  const plexes = createPlexKind(
    {
      workspace,
      ...(host.ui.commands ? { commands: host.ui.commands } : {}),
      slots: host.ui.slots,
      notes,
      relations,
      nesting: readNesting(plugin),
      names,
      look: readLook(plugin),
    },
    () =>
      usePlexView(
        { listRelatedNotes: (path, wanted) => notes.listRelatedNotes(path, wanted) },
        depth,
      ),
    PlexGraphView,
  )
  plugin.register.tabKind(plexes.kind)
  host.events.onVaultChange((event) => plexes.refresh(event))

  plugin.register.command({
    id: 'graph.open',
    handler: async (note) => {
      await host.tabs.navigate?.(PLEX, note.path)
    },
  })
}

export default definePlugin({
  activate,
})
