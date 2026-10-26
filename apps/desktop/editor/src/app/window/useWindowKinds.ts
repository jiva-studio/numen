/**
 * Tab kinds and openers declared for the desktop window.
 */
import { shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { MessageWriter } from '@/shared/notices/messages'
import type { DestinationDeps } from '@/features/command-palette'
import { AGENT, FILES, PLEX } from '@/entities/tab'
import { agentPort } from '../plugins/agentPort'
import type { WindowKindsDeps } from '../kinds/deps'
import {
  createAgentHost,
  createVaultNotes,
  createWorkspaceHost,
  type PluginServices,
} from '../plugins/pluginServices'
import { createFileMakers, createVaultFiles } from '../plugins/fileServices'
import { createBufferHost } from '../plugins/bufferHost'

export type { WindowKindsDeps } from '../kinds/deps'

function formatLocalDate(date: Date): string {
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${date.getFullYear()}-${month}-${day}`
}

/** Something to ask, put in the agent the person was last in and put in front. */
function createAgentAsker(windowTabs: WindowKindsDeps['windowTabs'], writeMessage: MessageWriter) {
  return {
    askQuestion: (text: string): void => {
      if (!windowTabs.hasKind(AGENT)) {
        writeMessage(
          'The agent chat is not available: its plugin is off or did not start.',
          'error',
        )
        return
      }
      windowTabs
        .sendText(AGENT, text)
        .catch((error: unknown) => writeMessage(formatErrorMessage(error), 'error'))
    },
  }
}

/**
 * Initializes and registers view kinds for the desktop window.
 */
export function useWindowKinds(deps: WindowKindsDeps) {
  const { core, log, tabOpeners, windowTabs, editing, settings } = deps
  const writeMessage = log.getWriter('command')
  const dragged = shallowRef<readonly string[]>([])

  /** A note put in front of the person in the plex they were last in, and in its editor where the window has no plex. */
  const travel = async (path: string): Promise<void> => {
    if (windowTabs.hasKind(PLEX)) await windowTabs.navigateTab(PLEX, path)
    else await tabOpeners.openFile(path, '', 'here')
  }

  /** The note the plex the person was last in relies on, which a talk is about. */
  const getPlexTarget = () => {
    const last = windowTabs.handle.last(PLEX)
    const target = last ? windowTabs.getTab(last.id)?.kind.getTarget?.(last.state) : undefined
    return { path: target?.path ?? '', title: target?.title ?? '' }
  }

  const agents = createAgentAsker(windowTabs, writeMessage)

  const destinations: DestinationDeps = {
    travel: (path) => travel(path),
    openFileAt: (path, run) => tabOpeners.openFileAt(path, [run]),
    openFile: (path, title, line) => void tabOpeners.openFile(path, title, { where: 'here', line }),
  }

  const createdItem = createFileMakers({
    tabOpeners,
    windowTabs,
    writeMessage,
  })

  windowTabs.registerKinds(settings.kinds)

  const services: PluginServices = {
    workspace: createWorkspaceHost({
      ...deps,
      nesting: settings.nesting,
      dragged,
      getNoteInFocus: getPlexTarget,
      ask: (text) => agents.askQuestion(text),
      writeMessage,
      getMessageWriter: (name) => log.getWriter(name),
      today: () => formatLocalDate(new Date()),
    }),
    notes: createVaultNotes({ core, search: core, making: editing.making }),
    files: createVaultFiles({ ...deps, made: createdItem, making: editing.making }),
    agent: createAgentHost({ port: agentPort, ...deps }),
    buffers: createBufferHost({ fileFlush: editing.fileFlush }),
  }

  return {
    agents,
    files: {
      revealPath: async (path: string): Promise<void> => {
        if (windowTabs.hasKind(FILES)) await windowTabs.navigateTab(FILES, path)
      },
    },
    made: createdItem,
    destinations,
    dragged,
    travel,
    leavePath: (from: string, to: string) => windowTabs.navigateTabsAt(from, to),
    services,
  }
}
