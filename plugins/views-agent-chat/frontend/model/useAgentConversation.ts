/**
 * Conversation state for agent tabs.
 */
import { computed, ref, shallowRef, watch, type ComputedRef, type Ref } from 'vue'
import type { AgentModelOption } from '../message-composer'
import type { Conversation } from '../thread'
import type { Turn } from '../types'
import {
  areLinkTargetsEqual,
  parseLinkTarget,
  parseVaultFileLink,
  extractLinkTargets,
} from '../lib/links'
import type { TextLink } from '@numen/plugin-sdk'
import type { AgentTabDeps } from '../types'
import type { SavedConversation } from '../lib/saved'

export type { AgentTabDeps } from '../types'

/** What one agent tab holds. */
export type AgentTabState = ReturnType<typeof useAgentConversation>

type ResolvedAddressMap = ReadonlyMap<string, string>

const AGENT_USE_AT = ['app', 'agent', 'use'] as const

const getModelAt = (agent: string): readonly string[] => ['app', 'agent', agent, 'model']

const getKeptValue = (deps: AgentTabDeps, path: readonly string[]): string =>
  String(deps.settings?.getSetting?.(path) ?? '')

function pickOption(
  options: readonly AgentModelOption[],
  selectedAgentId: string,
  ...[selectedModelId, deps]: [selectedModelId: string, deps: AgentTabDeps]
): AgentModelOption {
  const agent = selectedAgentId || getKeptValue(deps, AGENT_USE_AT)
  const model = selectedModelId || getKeptValue(deps, getModelAt(agent))
  const ofAgent = options.filter((o) => o.agentId === agent)
  return (
    ofAgent.find((o) => o.modelId === model) ??
    ofAgent.find((o) => o.isDefault) ??
    ofAgent[0] ??
    options[0]!
  )
}

function useAgentSelection(deps: AgentTabDeps) {
  const getModelsAt = (path: readonly string[]) => deps.settings?.getModelsAt?.(path) ?? []
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
          reasoningLevels: (one.reasoningLevels ?? []).map((level) => ({
            id: level.name,
            title: level.title,
          })),
          defaultReasoning: one.defaultReasoning ?? '',
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
  }

  const isAgentMissing = computed(() => modelOptions.value.length === 0)
  const openSettings = () => deps.settings?.openSettings?.()

  const models = { selectedAgentId, selectedModelId, modelOptions, selectModel }
  return {
    models,
    reasoning: useReasoningSelection(deps, models),
    agent: { isAgentMissing, openSettings, unreachable: deps.unreachable },
  }
}

function useReasoningSelection(
  deps: AgentTabDeps,
  selection: {
    readonly selectedAgentId: Ref<string>
    readonly selectedModelId: Ref<string>
    readonly modelOptions: ComputedRef<readonly AgentModelOption[]>
  },
) {
  const selected = ref({ agentId: '', id: '' })

  const selectedOption = computed(() =>
    selection.modelOptions.value.find(
      (one) =>
        one.agentId === selection.selectedAgentId.value &&
        one.modelId === selection.selectedModelId.value,
    ),
  )

  const reasoningLevels = computed(() => selectedOption.value?.reasoningLevels ?? [])

  /** The level chosen or kept, else the default of the model; empty where the model hasLevel none. */
  const selectedReasoningId = computed(() => {
    const agentId = selection.selectedAgentId.value
    const configuredReasoning =
      (selected.value.agentId === agentId ? selected.value.id : '') ||
      (deps.settings?.getReasoning?.() ?? '')
    const hasLevel = (id: string) => reasoningLevels.value.some((level) => level.id === id)
    if (hasLevel(configuredReasoning)) return configuredReasoning
    const byDefault = selectedOption.value?.defaultReasoning ?? ''
    return hasLevel(byDefault) ? byDefault : ''
  })

  const selectReasoningLevel = (id: string) => {
    const agentId = selection.selectedAgentId.value
    selected.value = { agentId, id }
  }

  return { reasoningLevels, selectedReasoningId, selectReasoningLevel }
}

const isSameList = (a: readonly string[], b: readonly string[]) =>
  a.length === b.length && a.every((one, index) => one === b[index])

function openResolvedOrDirect(deps: AgentTabDeps, vaultPath: string): void {
  const linkName = vaultPath.replace(/\.md$/i, '')
  deps
    .resolveLinks(`[[${linkName}]]`)
    .then((found) => {
      const resolvedPath = found[0]?.path || vaultPath
      deps.openFileAdjacent(resolvedPath)
    })
    .catch(() => {
      deps.openFileAdjacent(vaultPath)
    })
}

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
        ...[here, ...named].map(({ start, length }) => ({ start, end: start + length })),
      )
      return
    }
    const path = getResolved(href)
    if (path !== undefined) {
      if (path) deps.openFileAdjacent(path)
      return
    }
    const vaultPath = parseVaultFileLink(href)
    if (vaultPath) {
      press.preventDefault()
      openResolvedOrDirect(deps, vaultPath)
    }
  }
}

/** The wikilinks of one turn, read by the core from the turn's text. */
interface TurnLinks {
  readonly text: string
  readonly links: readonly TextLink[]
}

const NO_TEXT_LINKS: readonly TextLink[] = []

function useConversationAddresses(conversation: Conversation, deps: AgentTabDeps) {
  const resolvedAddresses = shallowRef<ResolvedAddressMap>(new Map())
  const linksByTurn = shallowRef<ReadonlyMap<string, TurnLinks>>(new Map())
  // A turn has at most one request in flight; a text that changes meanwhile is queried after it.
  const pendingTurns = new Set<string>()
  const enrichedByTurn = new WeakMap<
    Turn,
    { links: readonly TextLink[]; unresolved: readonly string[]; enriched: Turn }
  >()

  const resolveTurn = (turn: Turn) => {
    if (pendingTurns.has(turn.id) || linksByTurn.value.get(turn.id)?.text === turn.text) return
    // A text without `[[` holds no wikilink, and the core is not queried about it.
    if (!turn.text.includes('[[')) {
      linksByTurn.value = new Map(linksByTurn.value).set(turn.id, {
        text: turn.text,
        links: NO_TEXT_LINKS,
      })
      return
    }
    pendingTurns.add(turn.id)
    const { id, text } = turn
    deps
      .resolveLinks(text)
      .then((found) => {
        resolvedAddresses.value = new Map([
          ...resolvedAddresses.value,
          ...found.map((one) => [one.address, one.path] as const),
        ])
        linksByTurn.value = new Map(linksByTurn.value).set(id, { text, links: found })
      })
      .catch(() => {
        linksByTurn.value = new Map(linksByTurn.value).set(id, { text, links: NO_TEXT_LINKS })
      })
      .finally(() => {
        pendingTurns.delete(id)
        const current = conversation.turns.value.find((one) => one.id === id)
        if (current && current.text !== text) resolveTurn(current)
      })
  }

  watch(
    () => {
      const all = conversation.turns.value
      const last = all[all.length - 1]
      return [all.length, last?.id, last?.text, last?.state]
    },
    () => {
      for (const turn of conversation.turns.value) resolveTurn(turn)
    },
    { immediate: true },
  )

  // A turn keeps the links of the text last read while a longer text is queried,
  // so a link already rendered stays rendered as the response arrives.
  const enrichTurn = (turn: Turn): Turn => {
    const links = linksByTurn.value.get(turn.id)?.links ?? NO_TEXT_LINKS
    if (!links.length) return turn
    const unresolved = [
      ...new Set(
        links
          .filter((one) => resolvedAddresses.value.get(one.address) === '')
          .map((one) => one.address),
      ),
    ]
    const recognized = enrichedByTurn.get(turn)
    if (recognized?.links === links && isSameList(recognized.unresolved, unresolved))
      return recognized.enriched
    const enriched: Turn = {
      ...turn,
      unresolved,
      links: links.map((one) => ({ raw: one.raw, href: one.address, text: one.text })),
    }
    enrichedByTurn.set(turn, { links, unresolved, enriched })
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
    conversation
      .ask(text, '', {
        agentId: selection.models.selectedAgentId.value,
        modelId: selection.models.selectedModelId.value,
        reasoning: selection.reasoning.selectedReasoningId.value,
      })
      .catch(() => {
        userQuestion.value = text
      })
  }

  /** Shows the question and the turns a tab saved before a restart. */
  const restore = (saved: SavedConversation) => {
    userQuestion.value = saved.question
    if (saved.turns.length > 0) conversation.turns.value = saved.turns.map((turn) => ({ ...turn }))
  }

  const openTurnSource = (turn: Turn) => {
    const location = conversation.getSourceLocation(turn.id)
    if (location) deps.openFileAt(location.path, location.range)
  }

  return {
    conversation,
    models: selection.models,
    reasoning: selection.reasoning,
    agent: selection.agent,
    addresses,
    draft: { userQuestion, setQuestion, send, restore, openTurnSource },
  }
}
