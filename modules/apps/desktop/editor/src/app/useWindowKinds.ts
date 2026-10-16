/** The tab kinds a window offers, and what it keeps across them. */
import { ref } from 'vue'
import { createAgentKind } from './kinds/agent'
import { createReaderKinds } from './kinds/readers'
import { createFilesKind, type DestinationDeps } from './kinds/files'
import { createPlexKind } from './kinds/plex'
import type { WindowKindsDeps } from './kinds/deps'

export function useWindowKinds(deps: WindowKindsDeps) {
  const { held, editing, settings, writeMessage, tabOpeners } = deps
  const dragged = ref<readonly string[]>([])

  const plexes = createPlexKind({ ...deps, dragged })

  const agents = createAgentKind({
    ...deps,
    about: () => {
      const path = plexes.looking()
      return { path, title: plexes.getName(path) || path }
    },
  })

  const { read } = createReaderKinds(deps)

  const destinations: DestinationDeps = {
    travel: (path) => plexes.travel(path),
    openFileAt: (path, run) => tabOpeners.openFileAt(path, [run]),
    openFile: (path, title, line) => void tabOpeners.openFile(path, title, { where: 'here', line }),
  }

  const { files, made } = createFilesKind({ ...deps, dragged, writeMessage, destinations })

  held.registerKinds([
    ...editing.kinds,
    ...settings.kinds,
    plexes.kind,
    agents.kind,
    read.kind,
    files.kind,
  ])

  return {
    plexes,
    agents,
    read,
    files,
    made,
    destinations,
    dragged,
  }
}
