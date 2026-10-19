/**
 * Tab kinds and openers declared for the desktop window.
 */
import { shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { DestinationDeps } from '@/features/command-palette'
import { FILES, PLEX } from '@/entities/tab'
import { createAgentKind } from './kinds/agent'
import type { WindowKindsDeps } from './kinds/deps'
import { createVaultNotes, createWorkspaceHost, type PluginServices } from './pluginServices'
import { createFileMakers, createVaultFiles } from './fileServices'

export type { WindowKindsDeps } from './kinds/deps'

/**
 * Initializes and registers view kinds for the desktop window.
 */
export function useWindowKinds(deps: WindowKindsDeps) {
  const { core, log, tabOpeners, held, editing, settings } = deps
  const writeMessage = log.getWriter('command')
  const dragged = shallowRef<readonly string[]>([])

  /** A note put in front of the person in the plex they were last in, and in its editor where the window has no plex. */
  const travel = async (path: string): Promise<void> => {
    if (held.hasKind(PLEX)) await held.navigateTab(PLEX, path)
    else await tabOpeners.openFile(path, '', 'here')
  }

  /** The note the plex the person was last in stands on, which a talk is about. */
  const getPlexTarget = () => {
    const last = held.handle.last(PLEX)
    const target = last ? held.getTab(last.id)?.kind.getTarget?.(last.state) : undefined
    return { path: target?.path ?? '', title: target?.title ?? '' }
  }

  const agents = createAgentKind({ ...deps, about: getPlexTarget })

  const destinations: DestinationDeps = {
    travel: (path) => travel(path),
    openFileAt: (path, run) => tabOpeners.openFileAt(path, [run]),
    openFile: (path, title, line) => void tabOpeners.openFile(path, title, { where: 'here', line }),
  }

  const made = createFileMakers({ core, tabOpeners, notes: editing.notes, writeMessage })

  held.registerKinds([...editing.kinds, ...settings.kinds, agents.kind])

  const services: PluginServices = {
    workspace: createWorkspaceHost({
      ...deps,
      nesting: settings.nesting,
      dragged,
      ask: (text) => {
        agents
          .askQuestion(text)
          .catch((error: unknown) => writeMessage(formatErrorMessage(error), 'error'))
      },
      writeMessage,
    }),
    notes: createVaultNotes({ core, search: core, making: editing.making }),
    files: createVaultFiles({ ...deps, made, making: editing.making }),
  }

  return {
    agents,
    files: { revealPath: (path: string) => held.navigateTab(FILES, path) },
    made,
    destinations,
    dragged,
    travel,
    leavePath: (from: string, to: string) => held.navigateTabsAt(from, to),
    services,
  }
}
