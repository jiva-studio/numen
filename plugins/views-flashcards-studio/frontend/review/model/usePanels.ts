/**
 * The two panels beside the card, and which of the three views the window
 * displays.
 *
 * One view is displayed at a time, so a single state holds which one, and
 * opening a panel closes the other.
 */
import { computed, ref } from 'vue'

import { formatErrorMessage, type AgentPort, type TextLink, type Caught } from '@numen/plugin-sdk'
import { createCardAgentPort } from '../api/session/agent'
import { getDeckNeighborhood } from '../api/session/notes'
import { useAgentPanel } from './session/panel'
import { useNotesPanel } from './session/notes'
import type { PanelPlace } from './session/carousel'
import type { ReviewCard } from '../lib/card/cards'

/** What the panels ask of the window they stand in. */
export interface PanelsDeps {
  readonly agent: AgentPort
  readonly card: () => ReviewCard | null
  readonly unreachable: () => string
  readonly vault: () => string
  readonly showNotice: (text: string) => void
  readonly resolveLinks: (text: string) => Promise<readonly TextLink[]>
}

export const usePanels = (deps: PanelsDeps) => {
  /**
   * Which view the window displays. One view is displayed at a time, so a single
   * state holds which one, and opening a panel closes the other.
   */
  const showing = ref<'reading' | 'here' | 'asking'>('here')

  /** The same state expressed in the positions the strip uses for the three views. */
  const panelPlace = computed<PanelPlace>(() => {
    if (showing.value === 'reading') return 'before'
    if (showing.value === 'asking') return 'after'
    return 'here'
  })

  /**
   * Moves the strip to a position. A panel opened this way is opened and not
   * only displayed: its content is fetched and started when the panel is opened.
   */
  const reportFailure = (why: Caught): void => deps.showNotice(formatErrorMessage(why))

  const moveTo = (where: PanelPlace) => {
    if (where === 'before') notesPanel.actions.openPanel().catch(reportFailure)
    else if (where === 'after') agentPanel.openPanel()
    else showing.value = 'here'
  }

  const agentPanel = useAgentPanel({
    agent: (card) => createCardAgentPort(deps.agent, card, deps.vault()),
    card: deps.card,
    unreachable: deps.unreachable,
    open: () => showing.value === 'asking',
    // A panel put away takes the window back to the card only when the window is
    // on it: a card answered with the reading up ends the conversation, and the
    // reading keeps its position.
    showPanel: (open) => {
      if (open) showing.value = 'asking'
      else if (showing.value === 'asking') showing.value = 'here'
    },
    showNotice: deps.showNotice,
  })

  const notesPanel = useNotesPanel({
    open: () => showing.value === 'reading',
    showPanel: (open) => {
      if (open) showing.value = 'reading'
      else if (showing.value === 'reading') showing.value = 'here'
    },
    vault: deps.vault,
    deck: () => deps.card()?.address.deck ?? '',
    getDeckNeighborhood,
    showNotice: deps.showNotice,
    resolveLinks: deps.resolveLinks,
  })

  /**
   * The two entry points, each also the way out: a panel opened with a key or a
   * control is closed with the same one.
   *
   * A link pressed in the card opens the reading panel on the linked note.
   */
  const toggleNotes = (note = '') => {
    if (note === '' && showing.value === 'reading') notesPanel.actions.closePanel()
    else notesPanel.actions.openPanel(note).catch(reportFailure)
  }

  const toggleAgent = () => {
    if (showing.value === 'asking') agentPanel.closePanel()
    else agentPanel.openPanel()
  }

  return {
    showing,
    at: panelPlace,
    targetPanel: panelPlace,
    moveTo,
    agentPanel,
    notesPanel,
    toggleNotes,
    toggleAgent,
  }
}
