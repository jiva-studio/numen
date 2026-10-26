/**
 * The window an agent chat is drawn in, and the agent behind it, made up for a
 * test. Everything the chat asks of either is written down in the order asked.
 */
import { ref } from 'vue'
import type {
  AgentHost,
  AgentModel,
  NoteRef,
  OpenFileOptions,
  Span,
  WorkspaceHost,
} from '@numen/plugin-sdk'
import type { AgentChatHost } from '../model/tabDeps'

export interface HostOptions {
  /** The note the window relies on. */
  readonly focus?: NoteRef
  /** The vault path each written name stands for. */
  readonly notes?: Readonly<Record<string, string>>
  /** The models the settings list, by the path they are listed at. */
  readonly models?: Readonly<Record<string, readonly AgentModel[]>>
  /** The settings kept, by their path. */
  readonly settings?: Readonly<Record<string, unknown>>
  /** Why the agent cannot be reached. */
  readonly unreachable?: string
}

/** A window and an agent that answer a chat, and what the chat asked of them. */
export function createHost({
  focus = { path: '', title: '' },
  notes = {},
  models = {},
  settings = {},
  unreachable = '',
}: HostOptions = {}) {
  const recorded = {
    opened: [] as [string, OpenFileOptions | undefined][],
    openedAt: [] as [string, readonly Span[]][],
    settings: [] as [readonly string[], unknown][],
    finished: [] as string[],
    resolved: [] as (readonly string[])[],
    questions: [] as string[],
  }

  const request: Pick<
    WorkspaceHost,
    'openFile' | 'openFileAt' | 'getNoteInFocus' | 'isReady' | 'ask'
  > = {
    openFile: (path, options) => void recorded.opened.push([path, options]),
    ask: (text) => void recorded.questions.push(text),
    openFileAt: (path, spans) => void recorded.openedAt.push([path, spans]),
    getNoteInFocus: () => focus,
    isReady: ref(true),
  }
  const workspace = request as WorkspaceHost

  const agent: AgentHost = {
    ask: async function* () {},
    finish: async (conversation) => void recorded.finished.push(conversation),
    resolve: async (content) => {
      recorded.resolved.push(content)
      return new Map(content.filter((one) => notes[one]).map((one) => [one, notes[one]!]))
    },
    getUnreachable: () => unreachable,
    getSetting: (path) => settings[path.join('.')],
    writeSetting: async (path, value) => void recorded.settings.push([path, value]),
    getModelsAt: (path) => models[path.join('.')] ?? [],
  }

  const host: AgentChatHost = { workspace, agent }
  return { host, workspace, agent, recorded }
}
