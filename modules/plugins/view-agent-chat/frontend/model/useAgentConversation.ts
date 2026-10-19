/**
 * Conversation state for agent tabs.
 */
import { computed, ref, shallowRef, watch } from 'vue'
import {
  isNoteAddress,
  getWikilinks,
  type Conversation,
  type Turn,
  type AgentModelOption,
} from '@numen/ui'
import { areLinkTargetsEqual, parseLinkTarget, extractLinkTargets } from '../lib/links'
import type { AgentTabDeps } from '../types'

export { createTabTitle } from '../lib/title'
export type { AgentTabDeps } from '../types'

/** What one agent tab holds. */
export type AgentTabState = ReturnType<typeof useAgentConversation>

export type ResolvedAddressMap = ReadonlyMap<string, string>

const AGENT_USE_AT = ['agent', 'use'] as const

/** The path the models of one agent are named at. */
const getModelAt = (agent: string): readonly string[] => ['agent', agent, 'model']

/** The setting kept at a path, as text. */
const getKept = (deps: AgentTabDeps, path: readonly string[]): string =>
  String(deps.getSetting?.(path) ?? '')

/** The kept model of the kept agent, else that agent's default. */
function pickOption(
  options: readonly AgentModelOption[],
  selectedAgentId: string,
  ...[selectedModelId, deps]: [selectedModelId: string, deps: AgentTabDeps]
): AgentModelOption {
  const agent = selectedAgentId || getKept(deps, AGENT_USE_AT)
  const model = selectedModelId || getKept(deps, getModelAt(agent))
  const ofAgent = options.filter((o) => o.agentId === agent)
  return (
    ofAgent.find((o) => o.modelId === model) ??
    ofAgent.find((o) => o.isDefault) ??
    ofAgent[0] ??
    options[0]!
  )
}

function useAgentSelection(deps: AgentTabDeps) {
  const getModelsAt = (path: readonly string[]) => deps.getModelsAt?.(path) ?? []
  const getInstalledAgents = () =>
    getModelsAt(AGENT_USE_AT).filter((one) => one.name !== '' && one.presence === 'present')

  const modelOptions = computed<readonly AgentModelOption[]>(() =>
    getInstalledAgents().flatMap((agent) =>
      getModelsAt(getModelAt(agent.name))
        .filter((one) => one.name !== '' || one.isDefault)
        .map((one) => ({
          agentId: agent.name,
          agentTitle: agent.title,
          modelId: one.name,
          modelTitle: one.title,
          isAvailable: true,
          isDefault: one.isDefault,
        })),
    ),
  )

  const selectedAgentId = ref('')
  const selectedModelId = ref('')

  watch(
    modelOptions,
    (options) => {
      if (options[0] === undefined) {
        selectedAgentId.value = ''
        selectedModelId.value = ''
        return
      }
      const current = options.find(
        (o) => o.agentId === selectedAgentId.value && o.modelId === selectedModelId.value,
      )
      if (current) return
      const pick = pickOption(options, selectedAgentId.value, selectedModelId.value, deps)
      selectedAgentId.value = pick.agentId
      selectedModelId.value = pick.modelId
    },
    { immediate: true },
  )

  const selectModel = (option: AgentModelOption) => {
    selectedAgentId.value = option.agentId
    selectedModelId.value = option.modelId
    if (deps.writeSetting) {
      void deps.writeSetting(['agent', 'use'], option.agentId)
      void deps.writeSetting(['agent', option.agentId, 'model'], option.modelId)
    }
  }

  return { selectedAgentId, selectedModelId, modelOptions, selectModel }
}

const isSameList = (a: readonly string[], b: readonly string[]) =>
  a.length === b.length && a.every((one, index) => one === b[index])

function createConversationLinkFollower(
  deps: AgentTabDeps,
  getResolved: (href: string) => string | undefined,
) {
  return (turn: Turn, href: string, press: MouseEvent) => {
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
    const path = getResolved(href)
    if (path) deps.openFileBeside(path)
  }
}

function useConversationAddresses(conversation: Conversation, deps: AgentTabDeps) {
  const resolvedAddresses = shallowRef<ResolvedAddressMap>(new Map())
  const pendingAddresses = new Set<string>()
  const addressCache = new Map<string, { text: string; addresses: readonly string[] }>()
  const enrichedByTurn = new WeakMap<Turn, { unresolved: readonly string[]; enriched: Turn }>()

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

  const getTurnAddresses = (turn: Turn): readonly string[] => {
    const cached = addressCache.get(turn.id)
    if (cached && cached.text === turn.text) return cached.addresses
    const addresses = getWikilinks(turn.text).map((one) => one.address)
    addressCache.set(turn.id, { text: turn.text, addresses })
    return addresses
  }

  watch(
    () => {
      const all = conversation.turns.value
      const last = all[all.length - 1]
      return [all.length, last?.id, last?.text, last?.state]
    },
    () => resolveAddresses(conversation.turns.value.flatMap((turn) => getTurnAddresses(turn))),
    { immediate: true },
  )

  const enrichTurn = (turn: Turn): Turn => {
    const unresolved = getTurnAddresses(turn).filter(
      (address) => resolvedAddresses.value.get(address) === '',
    )
    if (!unresolved.length) return turn
    const known = enrichedByTurn.get(turn)
    if (known && isSameList(known.unresolved, unresolved)) return known.enriched
    const enriched = { ...turn, unresolved }
    enrichedByTurn.set(turn, { unresolved, enriched })
    return enriched
  }

  const turns = computed<Turn[]>(() => conversation.turns.value.map(enrichTurn))
  const followLink = createConversationLinkFollower(deps, (href) =>
    resolvedAddresses.value.get(href),
  )

  return { turns, followLink }
}

export function useAgentConversation(conversation: Conversation, deps: AgentTabDeps) {
  const userQuestion = ref('')
  const selection = useAgentSelection(deps)
  const addresses = useConversationAddresses(conversation, deps)

  const setQuestion = (text: string) => {
    userQuestion.value = text
  }

  const send = (text: string) => {
    userQuestion.value = ''
    void conversation.ask(text, '', {
      agentId: selection.selectedAgentId.value,
      modelId: selection.selectedModelId.value,
    })
  }

  const openTurnSource = (turn: Turn) => {
    const location = conversation.getSourceLocation(turn.id)
    if (location) deps.openFileAt(location.path, location.span)
  }

  return {
    ...conversation,
    ...selection,
    ...addresses,
    userQuestion,
    setQuestion,
    send,
    openTurnSource,
    unreachable: deps.unreachable,
  }
}
