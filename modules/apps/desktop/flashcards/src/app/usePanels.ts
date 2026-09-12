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
  readonly says: (said: string) => void
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
  const moved = (where: PanelPlace) => {
    if (where === 'before') void notesPanel.opens()
    else if (where === 'after') agentPanel.opens()
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
    shows: (open) => {
      if (open) showing.value = 'asking'
      else if (showing.value === 'asking') showing.value = 'here'
    },
    says: deps.says,
  })

  const notesPanel = useNotesPanel({
    open: () => showing.value === 'reading',
    shows: (open) => {
      if (open) showing.value = 'reading'
      else if (showing.value === 'reading') showing.value = 'here'
    },
    vault: deps.vault,
    deck: () => deps.card()?.deck ?? '',
    around,
    says: deps.says,
  })

  /**
   * The two ways in, each of them also the way out: a person who brought a panel
   * in with a key or a control takes it away with the same one.
   *
   * A link pressed in the card names the note to open on and brings the reading
   * in: the press was about that note.
   */
  const reads = (named = '') => {
    if (named === '' && showing.value === 'reading') notesPanel.shuts()
    else void notesPanel.opens(named)
  }

  const talks = () => {
    if (showing.value === 'asking') agentPanel.shuts()
    else agentPanel.opens()
  }

  return { showing, at, moved, agentPanel, notesPanel, reads, talks }
}
