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
  const { log, tabOpeners, held, editing, settings } = deps
  const writeMessage = log.getWriter('command')
  const dragged = shallowRef<readonly string[]>([])

  const plexes = createPlexKind({
    ...deps,
    dragged,
    writeMessage,
    askAgent: (text) => void agents.askQuestion(text),
  })

  const agents = createAgentKind({
    ...deps,
    about: () => {
      const path = plexes.looking()
      return { path, title: plexes.getName(path) || path }
    },
  })

  const { read, turned } = createReaderKinds(deps)

  const { recorded, pointed } = createMediaKinds(deps)

  const destinations: DestinationDeps = {
    travel: (path) => plexes.travel(path),
    openFileAt: (path, run) => tabOpeners.openFileAt(path, [run]),
    openFile: (path, title, line) => void tabOpeners.openFile(path, title, 'here', line),
  }

  const { files, made } = createFilesKind({ ...deps, dragged, writeMessage, destinations })

  held.registerKinds([
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
    destinations,
    dragged,
  }
}
