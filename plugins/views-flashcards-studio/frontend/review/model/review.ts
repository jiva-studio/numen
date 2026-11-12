/**
 * The review of one vault assembled: the screen that is shown is its state.
 *
 * The rules sit beside it: what a session is, what a keystroke requests, and
 * what the review reports. Everything here is created once, when the window
 * opens the review on a vault, and released when the window closes it.
 */
import { computed, onScopeDispose, shallowRef, watch, type ComputedRef, type Ref } from 'vue'

import { cards } from '../api/clients'
import { useScreens } from './screens'
import type { MountView } from './views'
import { getDeckName } from '../lib/vault/names'
import type { VaultCardsDue } from '../lib/vault/types'
import { useReviewDays } from './decks/reviewDays'
import { useVaultPresets } from './decks/presets'
import { useReviewSession } from './session/session'
import type { AgentHost, NoticeHost, Severity, Caught, Unvalidated } from '@numen/plugin-sdk'
import { createNotices } from './notices'
import type { VaultCounts } from './counts'
import { usePanels } from './usePanels'
import { createReviewKeys, type ReviewScreen } from './reviewKeys'
import type { Grade } from '../lib/card/grades'
import type { Report } from './session/session'

/**
 * What the review is given by the window it is opened in. The heatmap is drawn for the wider of the
 * screen and the page that `getViewport` reports, in pixels.
 */
export interface ReviewDeps {
  readonly vault: string
  readonly counts: VaultCounts
  readonly notices: NoticeHost
  readonly agent: AgentHost
  readonly mountView: MountView
  readonly goToVaults: () => void
  readonly getViewport: () => { readonly screen: number; readonly page: number }
}

/** The reading, as far as the keys reach into it. */
export interface NotesPage {
  scrollPage(back: boolean): void
}

/** Whether a ref callback received the notes panel, which exposes scrollPage. */
export const isNotesPage = (value: Unvalidated): value is NotesPage =>
  typeof value === 'object' &&
  value !== null &&
  'scrollPage' in value &&
  typeof value.scrollPage === 'function'

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

interface SessionActionDeps {
  vault: string
  goTo: (screen: ReviewScreen) => void
  state: ReturnType<typeof useReviewSession>
  agentPanel: { endConversation: () => void }
  showNotice: (text: string, severity: Severity) => void
}

function createSessionActions(deps: SessionActionDeps) {
  const openSession = async (deck: string, preset?: string) => {
    const report = await deps.state.opener.start(deps.vault, deck, preset)
    if (!report) return
    deps.goTo('session')
    reportSessionNotice(report, deps.showNotice)
  }

  const start = (deck: string) => openSession(deck)

  const startPreset = (preset: string) => openSession('', preset)

  const recordReview = async (how: Grade) => {
    deps.agentPanel.endConversation()
    await deps.state.answering.answer(how)
  }

  return { start, startPreset, recordReview }
}

interface NavigationActionDeps {
  vault: string
  counts: VaultCounts
  on: Ref<ReviewScreen>
  goTo: (screen: ReviewScreen) => void
  reviewDays: ReturnType<typeof useReviewDays>
  readSchedules: () => Promise<void>
  reportError: (error: Caught) => void
  goToVaults: () => void
}

function createNavigationActions(deps: NavigationActionDeps) {
  const leave = async () => {
    deps.goTo('decks')
    deps.reviewDays.read(deps.vault).catch(deps.reportError)
    await deps.counts.count()
    deps.readSchedules().catch(deps.reportError)
  }

  const goToVaults = () => {
    deps.counts.count().catch(deps.reportError)
    deps.goToVaults()
  }

  const refresh = async () => {
    if (deps.on.value === 'session') return
    await deps.counts.count()
    await deps.reviewDays.read(deps.vault)
    await deps.readSchedules()
  }

  return { leave, goToVaults, refresh }
}

function useChosenVault(deps: {
  vaultCount: ComputedRef<VaultCardsDue | null>
  counts: VaultCounts
  readSchedules: () => Promise<void>
  reportError: (error: Caught) => void
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
    widest: () => {
      const { screen, page } = deps.getViewport()
      return Math.max(screen, page)
    },
  })
  const schedules = useVaultPresets({ presets: cards })
  const notesPage = shallowRef<NotesPage | null>(null)

  const panels = usePanels({
    card: () => state.cards.queue.card.value,
    agent: deps.agent,
    unreachable: () => deps.agent.getUnreachable(),
    resolveLinks: (text) => deps.agent.resolveLinks(text, deps.vault),
    vault: () => deps.vault,
    showNotice: (text) => notices.showNotice(text, 'warning'),
  })

  const { on, goTo } = useScreens(['decks', 'session'] as const, {
    session: [
      state.opener.forget,
      panels.agentPanel.endConversation,
      panels.notesPanel.actions.endSession,
    ],
  })
  const vaultCount = computed(
    () => deps.counts.vaults.value.find((one) => one.identity.vault === deps.vault) ?? null,
  )

  const readSchedules = () => schedules.read(vaultCount.value, deps.counts.day.value)
  const sessionActions = createSessionActions({
    vault: deps.vault,
    goTo,
    state,
    agentPanel: panels.agentPanel,
    showNotice: notices.showNotice,
  })
  const navigationActions = createNavigationActions({
    vault: deps.vault,
    counts: deps.counts,
    on,
    goTo,
    reviewDays,
    readSchedules,
    reportError: notices.reportError,
    goToVaults: deps.goToVaults,
  })
  const actions = {
    readSchedules,
    start: sessionActions.start,
    startPreset: sessionActions.startPreset,
    leave: navigationActions.leave,
    goToVaults: navigationActions.goToVaults,
    recordReview: sessionActions.recordReview,
    refresh: navigationActions.refresh,
  }

  const { onKeyDown } = createReviewKeys({
    view: {
      getScreen: () => on.value,
      getChosenVault: () => vaultCount.value,
      getPresetsByDeck: () => schedules.byDeck.value,
      isShown: () => state.cards.queue.isAnswerShown.value,
      getShowing: () => panels.showing.value,
    },
    actions: {
      start: (deck) => {
        actions.start(deck).catch(notices.reportError)
      },
      show: state.show,
      answer: (how) => {
        actions.recordReview(how).catch(notices.reportError)
      },
      takeBack: () => {
        state.answering.takeBack().catch(notices.reportError)
      },
      leave: () => {
        actions.leave().catch(notices.reportError)
      },
      goToVaults: actions.goToVaults,
      scrollPage: (back) => notesPage.value?.scrollPage(back),
    },
    panels: {
      toggleAgent: panels.toggleAgent,
      toggleNotes: panels.toggleNotes,
      closeAgent: panels.agentPanel.closePanel,
      closeNotes: panels.notesPanel.actions.closePanel,
    },
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
    mountView: deps.mountView,
    on,
    vaultCount,
    onKeyDown,
    notesPage,
    actions,
    decks: { today: deps.counts.day, reviewDays, schedules },
    session: { state, panels },
  }
}

export type Review = ReturnType<typeof useReview>
