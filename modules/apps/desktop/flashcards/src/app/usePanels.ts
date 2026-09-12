/**
 * The two panels standing beside the card, and which of the three the window is
 * showing.
 *
 * One thing is in the window at a time, so one thing says which, and a panel
 * coming in is the other one going out.
 */
import { computed, ref } from 'vue'

import { around, core as agent, useAgentPanel, useNotesPanel } from '@/pages/session'
import type { PanelPlace } from '@/pages/session'
import type { CardFace } from '@/entities/card'

/** What the panels ask of the window they stand in. */
export interface PanelsDeps {
  /** The card in front of the person, and nothing between cards. */
  readonly card: () => CardFace | null
  /** Why nothing can be asked here, empty while something can. */
  readonly unreachable: () => string
  /** The vault the session is on. */
  readonly vault: () => string
  /** Where the window says what a person has to know. */
  readonly showNotice: (text: string) => void
}

export const usePanels = (deps: PanelsDeps) => {
  /**
   * Which of the card and the two panels beside it the window is showing. One
   * thing is in the window at a time, so one thing says which, and a panel coming
   * in is the other one going out.
   */
  const showing = ref<'reading' | 'here' | 'asking'>('here')

  /** The same thing in the words the strip stands the three in. */
  const at = computed<PanelPlace>(() =>
    showing.value === 'reading' ? 'before' : showing.value === 'asking' ? 'after' : 'here',
  )

  /**
   * The strip taken somewhere by a hand. A panel reached this way is opened, not
   * merely shown: what is in it is fetched and started when it is asked for.
   */
  const moveTo = (where: PanelPlace) => {
    if (where === 'before') void notesPanel.openPanel()
    else if (where === 'after') agentPanel.openPanel()
    else showing.value = 'here'
  }

  const agentPanel = useAgentPanel({
    agent,
    card: deps.card,
    unreachable: deps.unreachable,
    open: () => showing.value === 'asking',
    // A panel put away takes the window back to the card only when the window is
    // on it: a card answered with the reading up ends the conversation, and the
    // reading stays where it is.
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
    deck: () => deps.card()?.deck ?? '',
    around,
    showNotice: deps.showNotice,
  })

  /**
   * The two ways in, each of them also the way out: a person who brought a panel
   * in with a key or a control takes it away with the same one.
   *
   * A link pressed in the card names the note to open on and brings the reading
   * in: the press was about that note.
   */
  const toggleNotes = (note = '') => {
    if (note === '' && showing.value === 'reading') notesPanel.closePanel()
    else void notesPanel.openPanel(note)
  }

  const toggleAgent = () => {
    if (showing.value === 'asking') agentPanel.closePanel()
    else agentPanel.openPanel()
  }

  return { showing, at, moveTo, agentPanel, notesPanel, toggleNotes, toggleAgent }
}
