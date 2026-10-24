/**
 * The flashcards window put together: what it holds is which screen is on and
 * which vault is open.
 *
 * The rules are beside it: what a session is, what a keystroke asks for, what
 * the vaults come to, and what the window has to say. Everything here is made
 * once, as the window opens, and what comes out is placed under the screen it
 * belongs to.
 */
import { computed, ref, useTemplateRef, type ComputedRef, type Ref } from 'vue'

import { cards } from '@/shared/clients'
import { useScreens } from '@/shared/screens'
import { getDeckName, useReviewCounter, type VaultCardsDue } from '@/entities/vault'
import { useReviewDays, useVaultPresets } from '@/pages/decks'
import { useReviewSession } from '@/pages/session'
import type { Tone } from '@numen/ui'
import { useNotices } from './notices'
import { usePanels } from './usePanels'
import { createWindowKeys, type WindowScreen } from './windowKeys'
import { useWindowStreams } from './useWindowStreams'
import type { Grade } from '@/entities/card'
import type { NotesPanel, Report } from '@/pages/session'

/** Everything the window is made of, made once and handed to what draws it. */
const reportSessionNotice = (report: Report, showNotice: (text: string, tone: Tone) => void) => {
  if (report.unwritten.length) {
    showNotice(
      `Not asked from ${report.unwritten.map(getDeckName).join(', ')}: the deck could not be written.`,
      'caution',
    )
  }
  if (report.skipped > 0) {
    showNotice(`${report.skipped} answers in this vault could not be read.`, 'caution')
  }
}

interface WindowActionDeps {
  vault: Ref<string>
  chosen: ComputedRef<VaultCardsDue | null>
  today: Ref<string>
  on: Ref<WindowScreen>
  goTo: (screen: WindowScreen) => void
  stop: () => void
  count: () => Promise<void>
  reviewDays: ReturnType<typeof useReviewDays>
  schedules: ReturnType<typeof useVaultPresets>
  state: ReturnType<typeof useReviewSession>
  agentPanel: { endConversation: () => void; closePanel: () => void }
  showNotice: (text: string, tone: Tone) => void
}

function createWindowActions(deps: WindowActionDeps) {
  const choose = (id: string) => {
    deps.stop()
    deps.vault.value = id
    deps.goTo('decks')
    void deps.reviewDays.read(id)
    void deps.schedules.read(deps.chosen.value, deps.today.value)
  }

  const start = async (deck: string) => {
    const report = await deps.state.start(deps.vault.value, deck)
    if (!report) return
    deps.goTo('session')
    reportSessionNotice(report, deps.showNotice)
  }

  const startPreset = async (preset: string) => {
    const report = await deps.state.start(deps.vault.value, '', preset)
    if (!report) return
    deps.goTo('session')
    reportSessionNotice(report, deps.showNotice)
  }

  const leave = async () => {
    deps.goTo('decks')
    void deps.reviewDays.read(deps.vault.value)
    await deps.count()
    void deps.schedules.read(deps.chosen.value, deps.today.value)
  }

  const goToVaults = async () => {
    deps.goTo('vaults')
    await deps.count()
  }

  const answerCard = async (how: Grade) => {
    deps.agentPanel.endConversation()
    await deps.state.answer(how)
  }

  const refresh = async () => {
    if (deps.on.value === 'session') return
    await deps.count()
    await deps.reviewDays.read(deps.vault.value)
    await deps.schedules.read(deps.chosen.value, deps.today.value)
  }

  return { choose, start, startPreset, leave, goToVaults, answerCard, refresh }
}

function createWindowKeyboard(deps: {
  on: Ref<WindowScreen>
  vaults: Ref<readonly VaultCardsDue[]>
  chosen: ComputedRef<VaultCardsDue | null>
  schedules: ReturnType<typeof useVaultPresets>
  state: ReturnType<typeof useReviewSession>
  showing: Ref<'reading' | 'here' | 'asking'>
  actions: ReturnType<typeof createWindowActions>
  toggleAgent: () => void
  toggleNotes: () => void
  page: Readonly<Ref<InstanceType<typeof NotesPanel> | null>>
  agentPanel: { closePanel: () => void }
  notesPanel: { closePanel: () => void }
}) {
  return createWindowKeys({
    getScreen: () => deps.on.value,
    getVaults: () => deps.vaults.value,
    getChosenVault: () => deps.chosen.value,
    getPresetsByDeck: () => deps.schedules.byDeck.value,
    isShown: () => deps.state.isAnswerShown.value,
    getShowing: () => deps.showing.value,
    choose: deps.actions.choose,
    start: (deck) => void deps.actions.start(deck),
    show: deps.state.show,
    answer: (how) => void deps.actions.answerCard(how),
    takeBack: () => void deps.state.takeBack(),
    leave: () => void deps.actions.leave(),
    goToVaults: () => void deps.actions.goToVaults(),
    toggleAgent: deps.toggleAgent,
    toggleNotes: deps.toggleNotes,
    scrollPage: (back) => deps.page.value?.scrollPage(back),
    closeAgent: deps.agentPanel.closePanel,
    closeNotes: deps.notesPanel.closePanel,
  })
}

function useWindowScreens(deps: {
  reviewDays: ReturnType<typeof useReviewDays>
  schedules: ReturnType<typeof useVaultPresets>
  vault: Ref<string>
  state: ReturnType<typeof useReviewSession>
  agentPanel: ReturnType<typeof usePanels>['agentPanel']
  notesPanel: ReturnType<typeof usePanels>['notesPanel']
}) {
  return useScreens(['vaults', 'decks', 'session'] as const, {
    decks: [
      deps.reviewDays.forget,
      deps.schedules.forget,
      () => {
        deps.vault.value = ''
      },
    ],
    session: [deps.state.forget, deps.agentPanel.endConversation, deps.notesPanel.endSession],
  })
}

function useWindowState() {
  const vault = ref('')
  const notices = useNotices()
  const counter = useReviewCounter({ cards, reportError: notices.reportError })
  const state = useReviewSession({ cards, reportError: notices.reportError })
  const reviewDaysInstance = useReviewDays({
    cards,
    reportError: notices.reportError,
    widest: () => Math.max(window.screen.width, document.documentElement.clientWidth),
  })
  const schedules = useVaultPresets({ presets: cards })
  const unreachable = ref('')

  return { vault, notices, counter, state, reviewDays: reviewDaysInstance, schedules, unreachable }
}

function createWindowView(deps: {
  on: Ref<WindowScreen>
  notices: ReturnType<typeof useNotices>
  actions: ReturnType<typeof createWindowActions>
  counter: ReturnType<typeof useReviewCounter>
  reviewDays: ReturnType<typeof useReviewDays>
  schedules: ReturnType<typeof useVaultPresets>
  chosen: ComputedRef<VaultCardsDue | null>
  state: ReturnType<typeof useReviewSession>
  panels: ReturnType<typeof usePanels>
}) {
  return {
    on: deps.on,
    notices: deps.notices.notices,
    dismissNotice: deps.notices.dismissNotice,
    vaults: {
      list: deps.counter.vaults,
      isCounting: deps.counter.isCounting,
      choose: deps.actions.choose,
    },
    decks: {
      chosen: deps.chosen,
      today: deps.counter.day,
      reviewDays: deps.reviewDays,
      done: deps.reviewDays,
      schedules: deps.schedules,
      start: deps.actions.start,
      startPreset: deps.actions.startPreset,
      goToVaults: deps.actions.goToVaults,
    },
    session: {
      state: deps.state,
      at: deps.panels.at,
      moveTo: deps.panels.moveTo,
      answerCard: deps.actions.answerCard,
      toggleAgent: deps.panels.toggleAgent,
      toggleNotes: deps.panels.toggleNotes,
      leave: deps.actions.leave,
      agentPanel: deps.panels.agentPanel,
      notesPanel: deps.panels.notesPanel,
    },
  }
}

function useWindowListeners(deps: {
  on: Ref<WindowScreen>
  counter: ReturnType<typeof useReviewCounter>
  chosen: ComputedRef<VaultCardsDue | null>
  schedules: ReturnType<typeof useVaultPresets>
  state: ReturnType<typeof useReviewSession>
  panels: ReturnType<typeof usePanels>
  actions: ReturnType<typeof createWindowActions>
  page: Readonly<Ref<InstanceType<typeof NotesPanel> | null>>
  notices: ReturnType<typeof useNotices>
  unreachable: Ref<string>
}) {
  const { onKeyDown } = createWindowKeyboard({
    on: deps.on,
    vaults: deps.counter.vaults,
    chosen: deps.chosen,
    schedules: deps.schedules,
    state: deps.state,
    showing: deps.panels.showing,
    actions: deps.actions,
    toggleAgent: deps.panels.toggleAgent,
    toggleNotes: deps.panels.toggleNotes,
    page: deps.page,
    agentPanel: deps.panels.agentPanel,
    notesPanel: deps.panels.notesPanel,
  })

  useWindowStreams({
    reportError: deps.notices.reportError,
    setTasks: deps.notices.setTasks,
    onKeyDown,
    count: deps.counter.count,
    stop: deps.counter.stop,
    refresh: deps.actions.refresh,
    unreachable: deps.unreachable,
  })
}

export const useWindow = () => {
  const { vault, notices, counter, state, reviewDays, schedules, unreachable } = useWindowState()
  const panels = usePanels({
    card: () => state.card.value,
    unreachable: () => unreachable.value,
    vault: () => vault.value,
    showNotice: (text) => notices.showNotice(text, 'caution'),
  })

  const { on, goTo } = useWindowScreens({
    reviewDays,
    schedules,
    vault,
    state,
    agentPanel: panels.agentPanel,
    notesPanel: panels.notesPanel,
  })
  const page = useTemplateRef<InstanceType<typeof NotesPanel>>('page')
  const selected = computed(
    () => counter.vaults.value.find((one) => one.vault === vault.value) ?? null,
  )

  const actions = createWindowActions({
    vault,
    chosen: selected,
    today: counter.day,
    on,
    goTo,
    stop: counter.stop,
    count: counter.count,
    reviewDays,
    schedules,
    state,
    agentPanel: panels.agentPanel,
    showNotice: notices.showNotice,
  })

  useWindowListeners({
    on,
    counter,
    chosen: selected,
    schedules,
    state,
    panels,
    actions,
    page,
    notices,
    unreachable,
  })

  return createWindowView({
    on,
    notices,
    actions,
    counter,
    reviewDays,
    schedules,
    chosen: selected,
    state,
    panels,
  })
}
