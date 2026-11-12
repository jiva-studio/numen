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
  JsonValue,
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
  readonly settings?: Readonly<Record<string, JsonValue>>
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
    settings: [] as [readonly string[], JsonValue][],
    finished: [] as string[],
    resolved: [] as string[],
    questions: [] as string[],
    commands: [] as [string, CommandTarget | undefined][],
  }

  const request = {
    files: {
      open: (path: string, options?: OpenFileOptions) => {
        recorded.opened.push([path, options])
      },
      openAt: (path: string, spans: readonly TextRange[]) => {
        recorded.openedAt.push([path, spans])
      },
    },
    ask: (text: string) => {
      recorded.questions.push(text)
    },
    getNoteInFocus: () => focus,
    window: { isReady: ref(true) },
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
    listItems: async () => [],
    run: async (id, target) => {
      recorded.commands.push([id, target])
      return ''
    },
  }

  const host: AgentChatHost = { workspace, agent, commands, getReasoning: () => reasoning }
  return { host, workspace, agent, recorded }
}
