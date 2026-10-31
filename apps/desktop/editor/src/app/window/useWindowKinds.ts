/**
 * Tab kinds and openers declared for the desktop window.
 */
import { shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/plugin-sdk'
import { runInBackground } from '@numen/ui'
import type { MessageWriter } from '@/shared/notices/messages'
import type { DestinationDeps } from '@/features/command-palette'
import { agentPort } from '../plugins/agentPort'
import type { WindowKindsDeps } from '../kinds/deps'
import {
  createAgentHost,
  createVaultNotes,
  createVaultRelations,
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

/**
 * Text sent to the most recently active tab of the kind that accepts text, which
 * is brought to the front.
 */
function createAgentAsker(windowTabs: WindowKindsDeps['windowTabs'], writeMessage: MessageWriter) {
  return {
    askQuestion: (text: string): void => {
      const textKind = windowTabs.findKind((kind) => kind.receiveText !== undefined)
      if (!textKind) {
        writeMessage(
          'The agent chat is not available: its plugin is off or did not start.',
          'error',
        )
        return
      }
      windowTabs
        .sendText(textKind.kind, text)
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

  /** Shows a note where a plugin shows notes, and in the editor when no plugin does. */
  const travel = async (path: string): Promise<void> => {
    if (!(await deps.showNote(path))) await tabOpeners.openFile(path, '', 'current')
  }

  /**
   * The note the most recently active tool tab depends on, among the tools that
   * follow a note: those that navigate to a path and report the path they show.
   */
  const getNoteInFocus = () => {
    const last = [...windowTabs.tabs.value]
      .reverse()
      .map((one) => windowTabs.getTab(one.id))
      .find((one) => one?.kind.isTool && one.kind.navigate && one.kind.getTarget)
    const target = last?.kind.getTarget?.(last.state)
    return { path: target?.path ?? '', title: target?.title ?? '' }
  }

  const agents = createAgentAsker(windowTabs, writeMessage)

  const destinations: DestinationDeps = {
    travel: (path) => travel(path),
    openFile: (path, title, line, selector) =>
      runInBackground(
        tabOpeners.openFile(path, title, { where: 'current', line, selector }),
        (error) => writeMessage(formatErrorMessage(error), 'error'),
      ),
    findFileIcon: (path) => tabOpeners.findFileIcon(path),
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
      dragged,
      getNoteInFocus,
      ask: (text) => agents.askQuestion(text),
      writeMessage,
      getMessageWriter: (name) => log.getWriter(name),
      today: () => formatLocalDate(new Date()),
    }),
    notes: createVaultNotes({
      core,
      search: core,
      making: editing.making,
      pluginNotes: deps.pluginNotes,
    }),
    relations: createVaultRelations({ core }),
    files: createVaultFiles({ ...deps, making: editing.making }),
    agent: createAgentHost({ port: agentPort, ...deps }),
    buffers: createBufferHost({ fileFlush: editing.fileFlush }),
  }

  return {
    made: createdItem,
    destinations,
    dragged,
    travel,
    leavePath: (from: string, to: string) => windowTabs.navigateTabsAt(from, to),
    services,
  }
}
