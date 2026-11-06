/**
 * A window host and an agent double for rendering an agent chat in tests. Every
 * call the chat makes on either is recorded in call order.
 */
import { ref } from 'vue'
import type {
  AgentHost,
  AgentModel,
  CommandTarget,
  CommandsHost,
  NoteRef,
  OpenFileOptions,
  TextRange,
  WorkspaceHost,
} from '@numen/plugin-sdk'
import type { AgentChatHost } from '../model/tabDeps'
import { createLinkResolver } from './links'

export interface HostOptions {
  /** The note the window relies on. */
  readonly focus?: NoteRef
  /** The vault path for each written name. */
  readonly notes?: Readonly<Record<string, string>>
  /** The models the settings list, by the path they are listed at. */
  readonly models?: Readonly<Record<string, readonly AgentModel[]>>
  /** The settings kept, by their path. */
  readonly settings?: Readonly<Record<string, unknown>>
  /** The depth of reasoning the plugin setting keeps. */
  readonly reasoning?: string
  /** Why the agent cannot be reached. */
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
    commands: [] as [string, CommandTarget | undefined][],
  }

  const request: Pick<
    WorkspaceHost,
    'openFile' | 'openFileAt' | 'getNoteInFocus' | 'isReady' | 'ask'
  > = {
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

  const commands: CommandsHost = {
    getIds: () => [],
    watch: () => ({ dispose: () => undefined }),
    run: async (id, target) => {
      recorded.commands.push([id, target])
    },
  }

  const host: AgentChatHost = { workspace, agent, commands, getReasoning: () => reasoning }
  return { host, workspace, agent, recorded }
}
