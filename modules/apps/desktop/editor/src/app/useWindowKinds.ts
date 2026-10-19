/**
 * Tab kinds and openers declared for the desktop window.
 */
import { shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/wire'
import type { MessageWriter } from '@/shared/notices/messages'
import type { DestinationDeps } from '@/features/command-palette'
import { AGENT, FILES, PLEX } from '@/entities/tab'
import { agentPort } from './agentPort'
import type { WindowKindsDeps } from './kinds/deps'
import {
  createAgentHost,
  createVaultNotes,
  createWorkspaceHost,
  type PluginServices,
} from './pluginServices'
import { createFileMakers, createVaultFiles } from './fileServices'
import { createBufferHost } from './bufferHost'

export type { WindowKindsDeps } from './kinds/deps'

/** Something to ask, put in the agent the person was last in and put in front. */
function createAgentAsker(held: WindowKindsDeps['held'], writeMessage: MessageWriter) {
  return {
    askQuestion: (text: string): void => {
      if (!held.hasKind(AGENT)) {
        writeMessage(
          'The agent chat is not available: its plugin is off or did not start.',
          'error',
        )
        return
      }
      held
        .sendText(AGENT, text)
        .catch((error: unknown) => writeMessage(formatErrorMessage(error), 'error'))
    },
  }
}

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

  const agents = createAgentAsker(held, writeMessage)

  const destinations: DestinationDeps = {
    travel: (path) => travel(path),
    openFileAt: (path, run) => tabOpeners.openFileAt(path, [run]),
    openFile: (path, title, line) => void tabOpeners.openFile(path, title, { where: 'here', line }),
  }

  const made = createFileMakers({ core, tabOpeners, held, notes: editing.notes, writeMessage })

  held.registerKinds(settings.kinds)

  const services: PluginServices = {
    workspace: createWorkspaceHost({
      ...deps,
      nesting: settings.nesting,
      dragged,
      getNoteInFocus: getPlexTarget,
      ask: (text) => agents.askQuestion(text),
      writeMessage,
      getMessageWriter: (name) => log.getWriter(name),
      today: () => settings.dayBegins.day.value,
    }),
    notes: createVaultNotes({ core, search: core, making: editing.making }),
    files: createVaultFiles({ ...deps, made, making: editing.making }),
    agent: createAgentHost({ port: agentPort, ...deps }),
    buffers: createBufferHost({ fileFlush: editing.fileFlush }),
  }

  return {
    agents,
    files: {
      revealPath: async (path: string): Promise<void> => {
        if (held.hasKind(FILES)) await held.navigateTab(FILES, path)
      },
    },
    made,
    destinations,
    dragged,
    travel,
    leavePath: (from: string, to: string) => held.navigateTabsAt(from, to),
    services,
  }
}
