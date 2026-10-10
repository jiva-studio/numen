/**
 * Conversation state for agent tabs.
 */
import { computed, ref, shallowRef, watch } from 'vue'
import {
  isNoteAddress,
  wikilinksIn,
  type Conversation,
  type Turn,
  type AgentModelOption,
} from '@numen/ui'
import { areLinkTargetsEqual, parseLinkTarget, extractLinkTargets } from '@/entities/note'
import type { AgentTabDeps } from '../types'

export { firstLine } from '../lib/title'
export type { AgentTabDeps, NoteRef } from '../types'

export const DEFAULT_AGENT_OPTIONS: readonly AgentModelOption[] = [
  {
    agentId: 'antigravity',
    agentTitle: 'Antigravity',
    modelId: 'gemini-3.7-flash-high',
    modelTitle: 'Gemini 3.7 Flash',
    isAvailable: true,
    isDefault: true,
  },
  {
    agentId: 'antigravity',
    agentTitle: 'Antigravity',
    modelId: 'gemini-3.1-pro-high',
    modelTitle: 'Gemini 3.1 Pro',
    isAvailable: true,
  },
  {
    agentId: 'claude',
    agentTitle: 'Claude Code',
    modelId: 'sonnet',
    modelTitle: 'Sonnet 3.7',
    isAvailable: true,
  },
  {
    agentId: 'claude',
    agentTitle: 'Claude Code',
    modelId: 'haiku',
    modelTitle: 'Haiku 3.5',
    isAvailable: true,
  },
  {
    agentId: 'claude',
    agentTitle: 'Claude Code',
    modelId: 'opus',
    modelTitle: 'Opus 3',
    isAvailable: true,
  },
  {
    agentId: 'codex',
    agentTitle: 'OpenAI Codex',
    modelId: 'gpt-5.6-terra',
    modelTitle: 'GPT-5.6 Terra',
    isAvailable: true,
  },
  {
    agentId: 'codex',
    agentTitle: 'OpenAI Codex',
    modelId: 'gpt-5.6-luna',
    modelTitle: 'GPT-5.6 Luna',
    isAvailable: true,
  },
]

/** What one agent tab holds. */
export type AgentTabState = ReturnType<typeof useAgentConversation>

export type ResolvedAddressMap = ReadonlyMap<string, string>

function getDefaultModel(agent: string): string {
  if (agent === 'antigravity') {
    return 'gemini-3.7-flash-high'
  }
  if (agent === 'codex') {
    return 'gpt-5.6-terra'
  }
  return 'sonnet'
}

export function useAgentConversation(conversation: Conversation, deps: AgentTabDeps) {
  /** The question being written, until it is sent. */
  const userQuestion = ref('')
  const initialAgent = (deps.getSetting?.(['agent', 'use']) as string | undefined) || 'antigravity'
  const initialModel =
    (deps.getSetting?.(['agent', initialAgent, 'model']) as string | undefined) ||
    getDefaultModel(initialAgent)

  const selectedAgentId = ref(initialAgent)
  const selectedModelId = ref(initialModel)
  const modelOptions = ref<readonly AgentModelOption[]>(DEFAULT_AGENT_OPTIONS)

  const setQuestion = (text: string) => {
    userQuestion.value = text
  }

  const selectModel = (option: AgentModelOption) => {
    selectedAgentId.value = option.agentId
    selectedModelId.value = option.modelId
    if (deps.writeSetting) {
      void deps.writeSetting(['agent', 'use'], option.agentId)
      void deps.writeSetting(['agent', option.agentId, 'model'], option.modelId)
    }
  }

  /** A question sent. What the person has open the agent reads for itself. */
  const send = (text: string) => {
    userQuestion.value = ''
    void conversation.ask(text, '', selectedAgentId.value, selectedModelId.value)
  }

  /** A line about work pressed: where that call was working is put in front. */
  const openTurnSource = (turn: Turn) => {
    const location = conversation.getSourceLocation(turn.id)
    if (location) deps.openFileAt(location.path, location.span)
  }

  /**
   * Where each address an answer points at lands. An address that reaches
   * nothing is held as the empty path, which is what draws the link as not
   * resolving.
   */
  const resolvedAddresses = shallowRef<ResolvedAddressMap>(new Map())
  const pendingAddresses = new Set<string>()

  /** Every address an answer points at, resolved once each as they arrive. */
  const resolveAddresses = (addresses: readonly string[]) => {
    const fresh = addresses.filter((one) => !pendingAddresses.has(one))
    if (!fresh.length) return
    for (const one of fresh) pendingAddresses.add(one)
    void deps.resolve(fresh).then((found) => {
      const next = new Map(resolvedAddresses.value)
      for (const one of fresh) next.set(one, found.get(one) ?? '')
      resolvedAddresses.value = next
    })
  }

  const addressesIn = (text: string) => wikilinksIn(text).map((one) => one.address)

  watch(
    conversation.turns,
    (all) => resolveAddresses(all.flatMap((turn) => addressesIn(turn.text))),
    { deep: true },
  )

  /** The turns as the thread draws them, each saying which of its links reach nothing. */
  const turns = computed<Turn[]>(() =>
    conversation.turns.value.map((turn) => {
      const unresolved = addressesIn(turn.text).filter(
        (address) => resolvedAddresses.value.get(address) === '',
      )
      return unresolved.length ? { ...turn, unresolved } : turn
    }),
  )

  /**
   * A link inside an answer pressed. One naming a place in the vault opens it,
   * and the other places that answer names in the same document are lit with
   * it. One naming a note opens that note beside what the person is looking
   * at. Any other link is left to whatever would follow it.
   */
  const followLink = (turn: Turn, href: string, press: MouseEvent) => {
    const here = parseLinkTarget(href)
    if (here) {
      press.preventDefault()
      const named = extractLinkTargets(turn.text).filter(
        (target) => target.path === here.path && !areLinkTargetsEqual(target, here),
      )
      deps.openFileAt(
        here.path,
        ...[here, ...named].map(({ start, length }) => ({ from: start, to: start + length })),
      )
      return
    }
    if (!isNoteAddress(href)) return
    const path = resolvedAddresses.value.get(href)
    if (path) deps.openFileBeside(path)
  }

  return {
    ...conversation,
    turns,
    userQuestion,
    selectedAgentId,
    selectedModelId,
    modelOptions,
    selectModel,
    setQuestion,
    send,
    openTurnSource,
    followLink,
    unreachable: deps.unreachable,
  }
}
