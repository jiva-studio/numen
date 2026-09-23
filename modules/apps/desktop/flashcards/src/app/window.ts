/**
 * The flashcards window put together: what it holds is which screen is on and
 * which vault is open.
 *
 * The rules are beside it: what a session is, what a keystroke asks for, what
 * the vaults come to, and what the window has to say. Everything here is made
 * once, as the window opens, and what comes out stands under the screen it
 * belongs to.
 */
import { computed, ref, useTemplateRef } from 'vue'

import { cards } from '@/shared/clients'
import { useScreens } from '@/shared/screens'
import { deckName, useReviewCounter } from '@/entities/vault'
import { useReviewDays, useVaultPresets } from '@/pages/decks'
import { useReviewSession } from '@/pages/session'
import { useNotices } from './notices'
import { usePanels } from './usePanels'
import { createWindowKeys } from './windowKeys'
import { useWindowStreams } from './useWindowStreams'
import type { Grade } from '@/entities/card'
import type { NotesPanel, Report } from '@/pages/session'

/** Everything the window is made of, made once and handed to what draws it. */
export const useWindow = () => {
  /** The vault whose decks are open, and whose cards are being asked. */
  const vault = ref('')

  const { notices, showNotice, reportError, setTasks, dismissNotice } = useNotices()
  const { vaults, isCounting, day: today, count, stop } = useReviewCounter({ cards, reportError })
  const state = useReviewSession({ cards, reportError })
  const done = useReviewDays({
    cards,
    reportError,
    // The grid stands inside the page, so the page is what bounds it. A page
    // zoomed out is wider in the pixels a layout is measured in than the
    // screen is, and the screen alone would leave the widest grids short.
    widest: () => Math.max(window.screen.width, document.documentElement.clientWidth),
  })
  const schedules = useVaultPresets({ presets: cards })

  /** Why nothing can be asked here, empty while something can. */
  const unreachable = ref('')

  const { showing, at, moveTo, agentPanel, notesPanel, toggleNotes, toggleAgent } = usePanels({
    card: () => state.card.value,
    unreachable: () => unreachable.value,
    vault: () => vault.value,
    showNotice: (text) => showNotice(text, 'caution'),
  })

  /**
   * The three screens, and what each of them holds. Everything a screen took up
   * stands here beside it, which is the whole of what going back lets go of.
   */
  const { on, goTo } = useScreens(['vaults', 'decks', 'session'] as const, {
    decks: [
      done.forget,
      schedules.forget,
      () => {
        vault.value = ''
      },
    ],
    session: [state.forget, agentPanel.endConversation, notesPanel.endSession],
  })

  /** What is read, so the keys can scroll it: the caret is nowhere in it. */
  const page = useTemplateRef<InstanceType<typeof NotesPanel>>('page')

  const chosen = computed(() => vaults.value.find((one) => one.vault === vault.value) ?? null)

  /**
   * Into a vault. A vault is opened once it has been counted, so what the rest of
   * them come to is nobody's question any more and the counting is let go of.
   */
  const choose = (id: string) => {
    stop()
    vault.value = id
    goTo('decks')
    void done.read(id)
    void schedules.read(chosen.value, today.value)
  }

  /**
   * What the session could not act on, said once as it canOpen: a deck whose cards
   * could not be given marks holds cards this session does not ask, and a line of
   * the vault's answers that could not be read is a card standing where the rest
   * of its history left it.
   */
  const reportSession = (report: Report) => {
    if (report.unwritten.length) {
      showNotice(
        `Not asked from ${report.unwritten.map(deckName).join(', ')}: the deck could not be written.`,
        'caution',
      )
    }
    if (report.skipped > 0) {
      showNotice(`${report.skipped} answers in this vault could not be read.`, 'caution')
    }
  }

  const start = async (deck: string) => {
    const report = await state.start(vault.value, deck)
    if (!report) return
    goTo('session')
    reportSession(report)
  }

  /** Sit down to every deck one preset schedules, held to the budget it keeps. */
  const startPreset = async (preset: string) => {
    const report = await state.start(vault.value, '', preset)
    if (!report) return
    goTo('session')
    reportSession(report)
  }

  /**
   * Out of a session and back to the decks, with the counts as they now stand and
   * the days too: what a person just answered is part of what they have done.
   */
  const leave = async () => {
    goTo('decks')
    void done.read(vault.value)
    await count()
    void schedules.read(chosen.value, today.value)
  }

  /** Back to the vaults, which is where a person picks another collection. */
  const goToVaults = async () => {
    goTo('vaults')
    await count()
  }

  /**
   * The card answered. The conversation the panel was holding is over with the
   * card it was about.
   */
  const answerCard = async (how: Grade) => {
    agentPanel.endConversation()
    await state.answer(how)
  }

  const { onKeyDown } = createWindowKeys({
    getScreen: () => on.value,
    getVaults: () => vaults.value,
    getChosenVault: () => chosen.value,
    getPresetsByDeck: () => schedules.byDeck.value,
    isShown: () => state.shown.value,
    getShowing: () => showing.value,
    choose,
    start: (deck) => void start(deck),
    show: state.show,
    answer: (how) => void answerCard(how),
    takeBack: () => void state.takeBack(),
    leave: () => void leave(),
    goToVaults: () => void goToVaults(),
    toggleAgent,
    toggleNotes,
    scrollPage: (back) => page.value?.scrollPage(back),
    closeAgent: agentPanel.closePanel,
    closeNotes: notesPanel.closePanel,
  })

  /**
   * A deck written or a card changed underneath the window. A session is left
   * alone: its cards were laid out when it opened, and what a deck says now is
   * read at the next one.
   */
  const refresh = async () => {
    if (on.value === 'session') return
    await count()
    if (!vault.value) return
    await done.read(vault.value)
    await schedules.read(chosen.value, today.value)
  }

  useWindowStreams({ reportError, setTasks, onKeyDown, count, stop, refresh, unreachable })

  return {
    /** The window's own: which screen is on, and what it has to say. */
    on,
    notices,
    dismissNotice,

    /** The list of vaults, and the way into one. */
    vaults: { list: vaults, isCounting, choose },

    /** One vault's decks and presets, and the ways to sit down to them. */
    decks: { chosen, today, done, schedules, start, startPreset, goToVaults },

    /** The session, and the two panels standing beside the card. */
    session: {
      state,
      at,
      moveTo,
      answerCard,
      toggleAgent,
      toggleNotes,
      leave,
      agentPanel,
      notesPanel,
    },
  }
}
