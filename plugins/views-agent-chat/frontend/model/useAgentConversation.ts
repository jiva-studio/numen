/**
 * Conversation state for agent tabs.
 */
import { computed, ref, shallowRef, watch } from 'vue'
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
  const addresses = useConversationAddresses(conversation, deps)

  const setQuestion = (text: string) => {
    userQuestion.value = text
  }

  const send = (text: string) => {
    userQuestion.value = ''
    conversation.ask(text).catch(() => {
      userQuestion.value = text
    })
  }

  /** Shows the question and the turns a tab saved before a restart. */
  const restore = (saved: SavedConversation) => {
    userQuestion.value = saved.question
    if (saved.turns.length > 0) conversation.turns.value = saved.turns.map((turn) => ({ ...turn }))
  }

  const openTurnSource = (turn: Turn) => {
    const path = conversation.getSourcePath(turn.id)
    if (path) deps.openFile(path)
  }

  return {
    conversation,
    addresses,
    openSettings: deps.openSettings,
    draft: { userQuestion, setQuestion, send, restore, openTurnSource },
  }
}
