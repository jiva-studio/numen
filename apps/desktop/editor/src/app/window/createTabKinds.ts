/**
 * Tab kinds and openers declared for the desktop window.
 */
import { shallowRef } from 'vue'
import { getCurrentDate } from '@numen/ui'
import { formatError, type Caught } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { FileIcons } from '@/features/command-palette'
import { relations } from '@/entities/note'
import { agentPort } from '@/entities/plugin'
import type { WindowKindsDeps } from '../plugins/tabKindsDeps'
import {
  createAgentHost,
  createVaultNotes,
  createWorkspaceHost,
  type PluginServices,
} from '../plugins/pluginServices'
import { createFileMakers, createVaultFiles } from '../plugins/fileServices'
import { createBufferHost } from '../plugins/bufferHost'

export type { WindowKindsDeps } from '../plugins/tabKindsDeps'

function formatLocalDate(date: Date): string {
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${date.getFullYear()}-${month}-${day}`
}

function createAgentAsker(windowTabs: WindowKindsDeps['windowTabs'], writeMessage: MessageWriter) {
  return {
    askQuestion: (text: string): void => {
      const textKind = windowTabs.groups.findKind((kind) => kind.events?.receiveText !== undefined)
      if (!textKind) {
        writeMessage(
          'The agent chat is not available: its plugin is off or did not start.',
          'error',
        )
        return
      }
      windowTabs.groups
        .sendText(textKind.kind, text)
        .catch((error: Caught) => writeMessage(formatError(error), 'error'))
    },
  }
}

/**
 * Initializes and registers view kinds for the desktop window.
 */
export function createTabKinds(deps: WindowKindsDeps) {
  const { core, log, tabOpeners, windowTabs, editing, settings } = deps
  const writeMessage = log.getWriter('command')
  const dragged = shallowRef<readonly string[]>([])

  /**
   * The note the most recently active tool tab depends on, among the tools that
   * follow a note: those that navigate to a path and report the path they show.
   */
  const getNoteInFocus = () => {
    const last = [...windowTabs.tabs.tabs.value]
      .reverse()
      .map((one) => windowTabs.tabs.getTab(one.id))
      .find(
        (one) =>
          one?.kind.display?.isTool &&
          one.kind.address?.navigate !== undefined &&
          one.kind.address?.getTarget !== undefined,
      )
    const target = last?.kind.address?.getTarget?.(last.state)
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

  windowTabs.groups.registerKinds(settings.kinds)

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
    services,
  }
}
