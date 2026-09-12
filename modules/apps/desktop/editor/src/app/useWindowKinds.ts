/**
 * Tab kinds and openers declared for the desktop window.
 */
import { shallowRef } from 'vue'
import type { DestinationDeps } from '@/features/command-palette'
import { createAgentKind } from './kinds/agent'
import { createFilesKind } from './kinds/files'
import { createMediaKinds } from './kinds/media'
import { createPlexKind } from './kinds/plex'
import { createReaderKinds } from './kinds/readers'
import type { WindowKindsDeps } from './kinds/deps'

export type { WindowKindsDeps } from './kinds/deps'

/**
 * Initializes and registers view kinds for the desktop window.
 */
export function useWindowKinds(deps: WindowKindsDeps) {
  const { log, puts, held, editing, settings } = deps
  const told = log.under('command')
  const dragged = shallowRef<readonly string[]>([])

  const plexes = createPlexKind({
    ...deps,
    dragged,
    told,
    asks: (text) => void agents.askQuestion(text),
  })

  const agents = createAgentKind({
    ...deps,
    about: () => {
      const path = plexes.looking()
      return { path, title: plexes.names(path) || path }
    },
  })

  const { read, turned } = createReaderKinds(deps)

  const { recorded, pointed } = createMediaKinds(deps)

  const places: DestinationDeps = {
    travel: (path) => plexes.travel(path),
    opensAt: (path, run) => puts.opensAt(path, [run]),
    opens: (path, title, line) => void puts.opens(path, title, 'here', line),
  }

  const { files, made } = createFilesKind({ ...deps, dragged, told, places })

  held.declares([
    ...editing.kinds,
    ...settings.kinds,
    plexes.kind,
    agents.kind,
    read.kind,
    turned.kind,
    recorded.kind,
    pointed.kind,
    files.kind,
  ])

  return {
    plexes,
    agents,
    read,
    turned,
    recorded,
    pointed,
    files,
    made,
    places,
    dragged,
  }
}
