/**
 * A window host and an agent double for rendering an agent chat in tests. Every
 * call the chat makes on either is recorded in call order.
 */
import { ref } from 'vue'
import type {
  AgentHost,
  AgentModel,
  NoteRef,
  OpenFileOptions,
  TextRange,
  WorkspaceHost,
} from '@numen/plugin-sdk'
import type { AgentChatHost } from '../model/tabDeps'
import { createLinkResolver } from './links'

export interface HostOptions {
  readonly focus?: NoteRef
  readonly notes?: Readonly<Record<string, string>>
  readonly models?: Readonly<Record<string, readonly AgentModel[]>>
  readonly settings?: Readonly<Record<string, unknown>>
  readonly reasoning?: string
  readonly unreachable?: string
}

/** A host and an agent that serve a chat, and the calls the chat made on them. */
export function createHost({
  focus = { path: '', title: '' },
  notes = {},
  models = {},
  settings = {},
  unreachable = '',
  reasoning = '',
}: HostOptions = {}) {
  const recorded = {
    opened: [] as [string, OpenFileOptions | undefined][],
    openedAt: [] as [string, readonly TextRange[]][],
    settings: [] as [readonly string[], unknown][],
    finished: [] as string[],
    resolved: [] as string[],
    questions: [] as string[],
    commands: [] as [string, NoteRef][],
  }

  const request: Pick<
    WorkspaceHost,
    'openFile' | 'openFileAt' | 'getNoteInFocus' | 'isReady' | 'ask' | 'runCommand'
  > = {
    runCommand: (id, target) => {
      recorded.commands.push([id, target])
    },
    openFile: (path, options) => {
      recorded.opened.push([path, options])
    },
    ask: (text) => {
      recorded.questions.push(text)
    },
    openFileAt: (path, spans) => {
      recorded.openedAt.push([path, spans])
    },
    getNoteInFocus: () => focus,
    isReady: ref(true),
  }
  const workspace = request as WorkspaceHost

  const agent: AgentHost = {
    ask: async function* () {
      yield* []
    },
    finish: async (conversation) => {
      recorded.finished.push(conversation)
    },
    resolveLinks: async (text) => {
      recorded.resolved.push(text)
      return createLinkResolver(notes)(text)
    },
    getUnreachable: () => unreachable,
    getSetting: (path) => settings[path.join('.')],
    writeSetting: async (path, value) => {
      recorded.settings.push([path, value])
    },
    getModelsAt: (path) => models[path.join('.')] ?? [],
  }

  const host: AgentChatHost = { workspace, agent, getReasoning: () => reasoning }
  return { host, workspace, agent, recorded }
}
