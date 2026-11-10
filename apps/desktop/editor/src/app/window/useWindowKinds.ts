/**
 * Tab kinds and openers declared for the desktop window.
 */
import { shallowRef } from 'vue'
import { formatErrorMessage } from '@numen/core-client'
import { getCurrentDate } from '@numen/ui'
import type { MessageWriter } from '@/shared/notices/messages'
import type { FileIcons } from '@/features/command-palette'
import { relations } from '@/entities/note'
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

  /** Opens the note in a tab of the current window. */
  const travel = async (path: string): Promise<void> => {
    await tabOpeners.openFile(path, '', 'current')
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

  const destinations: FileIcons = {
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
      today: () => formatLocalDate(getCurrentDate()),
    }),
    notes: createVaultNotes({
      search: core,
      making: editing.making,
      pluginNotes: deps.pluginNotes,
    }),
    relations,
    files: createVaultFiles({ ...deps, making: editing.making }),
    agent: createAgentHost({ port: agentPort, ...deps }),
    buffers: createBufferHost({ fileFlush: editing.fileFlush }),
    noteBuffer: editing.notes,
  }

  return {
    made: createdItem,
    destinations,
    dragged,
    travel,
    leavePath: (from: string, to: string) =>
      windowTabs.navigateTabsAt(from, to, (tabId) => deps.tabSync?.setShownPath(tabId, to)),
    services,
  }
}
