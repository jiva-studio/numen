/**
 * A window host and the links of a vault for rendering an agent chat in tests. Every call the chat
 * makes on them is recorded in call order.
 */
import { ref } from 'vue'
import type {
  NoteRef,
  OpenFileOptions,
  TextRange,
  VaultRelations,
  WorkspaceHost,
} from '@numen/plugin-sdk'
import type { AgentChatHost } from '../model/tabDeps'
import { createLinkResolver } from './links'

export interface HostOptions {
  readonly focus?: NoteRef
  readonly notes?: Readonly<Record<string, string>>
}

/** A host that serves a chat, and the calls the chat made on it. */
export function createHost({ focus = { path: '', title: '' }, notes = {} }: HostOptions = {}) {
  const recorded = {
    opened: [] as [string, OpenFileOptions | undefined][],
    openedAt: [] as [string, readonly TextRange[]][],
    resolved: [] as string[],
    settingsOpened: 0,
  }

  const workspace: WorkspaceHost = {
    window: {
      isReady: ref(true),
      openingPath: ref(''),
      readOpeningPath: async () => '',
      getToday: () => '2026-01-01',
    },
    dragged: { paths: ref([]), setPaths: () => undefined },
    files: {
      open: (path: string, options?: OpenFileOptions) => {
        recorded.opened.push([path, options])
      },
      openAt: (path: string, spans: readonly TextRange[]) => {
        recorded.openedAt.push([path, spans])
      },
      syncPath: () => false,
    },
    messages: { showError: () => undefined, getWriter: () => () => undefined },
    icons: { getCommand: () => null, getEntry: () => ({}) },
    getNoteInFocus: () => focus,
  }

  const resolve = createLinkResolver(notes)
  const relations = {
    resolveLinks: async (text: string) => {
      recorded.resolved.push(text)
      return resolve(text)
    },
  } satisfies Pick<VaultRelations, 'resolveLinks'>

  const host: AgentChatHost = {
    workspace,
    relations,
    openSettings: () => {
      recorded.settingsOpened += 1
    },
  }
  return { host, workspace, relations, recorded }
}
