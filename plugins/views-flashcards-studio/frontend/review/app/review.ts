/**
 * The review of one vault assembled: the screen that is shown is its state.
 *
 * The rules sit beside it: what a session is, what a keystroke requests, and
 * what the review reports. Everything here is created once, when the window
 * opens the review on a vault, and released when the window closes it.
 */
import { computed, onScopeDispose, shallowRef, watch, type ComputedRef, type Ref } from 'vue'

import { cards } from '../shared/clients'
import { useScreens } from '../shared/screens'
import type { MountView } from '../shared/views'
import { getDeckName, type VaultCardsDue } from '../entities/vault'
import { useReviewDays, useVaultPresets } from '../pages/decks'
import { useReviewSession } from '../pages/session'
import type { AgentHost, NoticeHost, Severity } from '@numen/plugin-sdk'
import { createNotices } from './notices'
import type { VaultCounts } from './counts'
import { usePanels } from './usePanels'
import { createReviewKeys, type ReviewScreen } from './reviewKeys'
import type { Grade } from '../entities/card'
import type { Report } from '../pages/session'

/** What the review is given by the window it is opened in. */
export interface ReviewDeps {
  /** The vault the window opened the review on. */
  readonly vault: string
  /** The cards due in every vault, from which the review of one vault reads its own. */
  readonly counts: VaultCounts
  /** The corner of the window the notices are drawn in. */
  readonly notices: NoticeHost
  /** The agent of the window, which reports whether it is reachable and answers questions about a card. */
  readonly agent: AgentHost
  /** How the agent panel mounts the conversation another plugin declares. */
  readonly mountView: MountView
  /** Out of the vault: the window closes the review and offers its vaults again. */
  readonly goToVaults: () => void
}

/** The reading, as far as the keys reach into it. */
export interface NotesPage {
  scrollPage(back: boolean): void
}

const reportSessionNotice = (
  report: Report,
  showNotice: (text: string, severity: Severity) => void,
) => {
  if (report.unwritten.length) {
    showNotice(
      `Not asked from ${report.unwritten.map(getDeckName).join(', ')}: the deck could not be written.`,
      'warning',
    )
  }
  if (report.skipped > 0) {
    showNotice(`${report.skipped} answers in this vault could not be read.`, 'warning')
  }
}

interface ReviewActionDeps {
  vault: string
  vaultCount: ComputedRef<VaultCardsDue | null>
  counts: VaultCounts
  on: Ref<ReviewScreen>
  goTo: (screen: ReviewScreen) => void
  reviewDays: ReturnType<typeof useReviewDays>
  schedules: ReturnType<typeof useVaultPresets>
  state: ReturnType<typeof useReviewSession>
  agentPanel: { endConversation: () => void }
  showNotice: (text: string, severity: Severity) => void
  reportError: (why: unknown) => void
  goToVaults: () => void
}

function createReviewActions(deps: ReviewActionDeps) {
  const readSchedules = () => deps.schedules.read(deps.vaultCount.value, deps.counts.day.value)

  const openSession = async (deck: string, preset?: string) => {
    const report = await deps.state.start(deps.vault, deck, preset)
    if (!report) return
    deps.goTo('session')
    reportSessionNotice(report, deps.showNotice)
  }

  const start = (deck: string) => openSession(deck)

  const startPreset = (preset: string) => openSession('', preset)

  const leave = async () => {
    deps.goTo('decks')
    deps.reviewDays.read(deps.vault).catch(deps.reportError)
    await deps.counts.count()
    readSchedules().catch(deps.reportError)
  }

  const goToVaults = () => {
    deps.counts.count().catch(deps.reportError)
    deps.goToVaults()
  }

  const recordReview = async (how: Grade) => {
    deps.agentPanel.endConversation()
    await deps.state.answer(how)
  }

  const refresh = async () => {
    if (deps.on.value === 'session') return
    await deps.counts.count()
    await deps.reviewDays.read(deps.vault)
    await readSchedules()
  }

  return { readSchedules, start, startPreset, leave, goToVaults, recordReview, refresh }
}

/**
 * The vault counted is the vault the decks are drawn from. Its presets are read
 * once it is, and the counting of the other vaults ends there: they are not
 * drawn again until the review is left.
 */
function useChosenVault(deps: {
  vaultCount: ComputedRef<VaultCardsDue | null>
  counts: VaultCounts
  readSchedules: () => Promise<void>
  reportError: (why: unknown) => void
}) {
  let isRead = false
  watch(
    () => deps.vaultCount.value?.isCounted ?? false,
    (isCounted) => {
      if (!isCounted || isRead) return
      isRead = true
      deps.counts.stop()
      deps.readSchedules().catch(deps.reportError)
    },
    { immediate: true },
  )
}

export const useReview = (deps: ReviewDeps) => {
  const notices = createNotices(deps.notices)
  const state = useReviewSession({ cards, reportError: notices.reportError })
  const reviewDays = useReviewDays({
    cards,
    reportError: notices.reportError,
    widest: () => Math.max(window.screen.width, document.documentElement.clientWidth),
  })
  const schedules = useVaultPresets({ presets: cards })
  const notesPage = shallowRef<NotesPage | null>(null)

  const panels = usePanels({
    card: () => state.card.value,
    agent: deps.agent,
    unreachable: () => deps.agent.getUnreachable(),
    resolveLinks: (text) => deps.agent.resolveLinks(text, deps.vault),
    vault: () => deps.vault,
    showNotice: (text) => notices.showNotice(text, 'warning'),
  })

  const { on, goTo } = useScreens(['decks', 'session'] as const, {
    session: [state.forget, panels.agentPanel.endConversation, panels.notesPanel.endSession],
  })
  const vaultCount = computed(
    () => deps.counts.vaults.value.find((one) => one.vault === deps.vault) ?? null,
  )

  const actions = createReviewActions({
    vault: deps.vault,
    vaultCount,
    counts: deps.counts,
    on,
    goTo,
    reviewDays,
    schedules,
    state,
    agentPanel: panels.agentPanel,
    showNotice: notices.showNotice,
    reportError: notices.reportError,
    goToVaults: deps.goToVaults,
  })

  const { onKeyDown } = createReviewKeys({
    getScreen: () => on.value,
    getChosenVault: () => vaultCount.value,
    getPresetsByDeck: () => schedules.byDeck.value,
    isShown: () => state.isAnswerShown.value,
    getShowing: () => panels.showing.value,
    start: (deck) => {
      actions.start(deck).catch(notices.reportError)
    },
    show: state.show,
    answer: (how) => {
      actions.recordReview(how).catch(notices.reportError)
    },
    takeBack: () => {
      state.takeBack().catch(notices.reportError)
    },
    leave: () => {
      actions.leave().catch(notices.reportError)
    },
    goToVaults: actions.goToVaults,
    toggleAgent: panels.toggleAgent,
    toggleNotes: panels.toggleNotes,
    scrollPage: (back) => notesPage.value?.scrollPage(back),
    closeAgent: panels.agentPanel.closePanel,
    closeNotes: panels.notesPanel.closePanel,
  })

  deps.counts.start()
  deps.counts.setRefresh(actions.refresh)
  reviewDays.read(deps.vault).catch(notices.reportError)
  useChosenVault({
    vaultCount,
    counts: deps.counts,
    readSchedules: actions.readSchedules,
    reportError: notices.reportError,
  })

  onScopeDispose(() => {
    deps.counts.setRefresh(null)
    goTo('decks')
  })

  return {
    vault: deps.vault,
    mountView: deps.mountView,
    on,
    vaultCount,
    onKeyDown,
    notesPage,
    decks: {
      today: deps.counts.day,
      reviewDays,
      schedules,
      start: actions.start,
      startPreset: actions.startPreset,
      goToVaults: actions.goToVaults,
    },
    session: {
      state,
      at: panels.at,
      moveTo: panels.moveTo,
      recordReview: actions.recordReview,
      toggleAgent: panels.toggleAgent,
      toggleNotes: panels.toggleNotes,
      leave: actions.leave,
      agentPanel: panels.agentPanel,
      notesPanel: panels.notesPanel,
    },
  }
}

export type Review = ReturnType<typeof useReview>
