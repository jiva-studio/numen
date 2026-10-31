/**
 * One session: the cards remaining, the position within them, and what has been
 * written into the vault.
 *
 * Separate from the template because these rules decide which cards are presented
 * and what is written to the vault, and a rule inside a component can only be
 * exercised through the screen.
 */
import { getNow } from '@numen/ui'
import { computed, ref, shallowRef } from 'vue'
import type { Ref } from 'vue'

import { RATING_BY_GRADE } from '../../../entities/card'
import type { Rating } from '../../../../api/types'
import type { ReviewCard, Grade, Intervals } from '../../../entities/card'

/** What a session requests of the application, and nothing more. */
export interface SessionClient {
  /**
   * Deck is one deck, or empty for every deck the vault holds. Preset holds it
   * to the decks one preset schedules, and to the budget that preset keeps.
   */
  startSession(params: { vault: string; deck: string; preset?: string }): Promise<SessionStart>
  recordReview(params: {
    vault: string
    run: string
    cardId: string
    template: string
    rating: Rating
    durationMs: number
  }): Promise<{ answer: string }>
  undoReview(params: { vault: string; run: string; answer: string }): Promise<unknown>
}

/** What opening a session comes back with. */
export interface SessionStart {
  run: string
  cards: readonly {
    deck: string
    section: string
    cardId: string
    template: string
    heading: string
    front: string
    back: string
    isNew: boolean
    intervals?: { again: number; hard: number; good: number; easy: number } | undefined
  }[]
  unwritten: readonly string[]
  skipped: number
}

/** What a session could not act on, for the window to say once as it opens. */
export interface Report {
  /** The decks holding a card that could not be given a card id. */
  readonly unwritten: readonly string[]
  /** How many lines of the vault's answers could not be read. */
  readonly skipped: number
}

/** What a session is built over: the application client, and the error reporter. */
export interface SessionDeps {
  cards: SessionClient
  reportError(why: unknown): void
  /** The current time in milliseconds. The time a card is on screen is measured with it. */
  now?(): number
}

function useSessionCards() {
  const request = shallowRef<readonly ReviewCard[]>([])
  const cardIndex = ref(0)
  const isAnswerShown = ref(false)
  const answers = ref<string[]>([])

  const card = computed<ReviewCard | null>(() => request.value[cardIndex.value] ?? null)
  const remaining = computed(() => request.value.length - cardIndex.value)
  const isOver = computed(() => card.value === null)
  const isDone = computed(() => answers.value.length)

  const reset = () => {
    request.value = []
    cardIndex.value = 0
    isAnswerShown.value = false
    answers.value = []
  }

  const api = {
    cardList: request,
    cardIndex,
    isAnswerShown,
    answers,
    card,
    left: remaining,
    isOver,
    completedCount: isDone,
    reset,
  }
  return api
}

async function openSession(deps: {
  cards: SessionClient
  reportError: (why: unknown) => void
  id: string
  deck: string
  preset?: string
}): Promise<{ run: string; cards: readonly ReviewCard[]; report: Report } | null> {
  try {
    const startedSession = await deps.cards.startSession(
      deps.preset === undefined
        ? { vault: deps.id, deck: deps.deck }
        : { vault: deps.id, deck: deps.deck, preset: deps.preset },
    )
    return {
      run: startedSession.run,
      cards: startedSession.cards.map(createReviewCard),
      report: { unwritten: startedSession.unwritten, skipped: startedSession.skipped },
    }
  } catch (why) {
    deps.reportError(why)
    return null
  }
}

async function submitAnswer(deps: {
  cards: SessionClient
  reportError: (why: unknown) => void
  vault: string
  run: string
  card: ReviewCard
  how: Grade
  elapsedMs: number
}): Promise<string | null> {
  try {
    const response = await deps.cards.recordReview({
      vault: deps.vault,
      run: deps.run,
      cardId: deps.card.cardId,
      template: deps.card.template,
      rating: RATING_BY_GRADE[deps.how],
      durationMs: deps.elapsedMs,
    })
    return response.answer
  } catch (why) {
    deps.reportError(why)
    return null
  }
}

async function revertAnswer(deps: {
  cards: SessionClient
  reportError: (why: unknown) => void
  vault: string
  run: string
  answer: string
}): Promise<boolean> {
  try {
    await deps.cards.undoReview({ vault: deps.vault, run: deps.run, answer: deps.answer })
    return true
  } catch (why) {
    deps.reportError(why)
    return false
  }
}

function createAnswerActions(deps: {
  cardsClient: SessionClient
  reportError: (why: unknown) => void
  now: () => number
  vault: Ref<string>
  run: Ref<string>
  cards: ReturnType<typeof useSessionCards>
  writing: Ref<boolean>
  getPut: () => number
  setPut: (time: number) => void
}) {
  const answer = async (how: Grade) => {
    const one = deps.cards.card.value
    if (!one || !deps.cards.isAnswerShown.value || deps.writing.value) return
    const elapsedMs = deps.now() - deps.getPut()
    deps.writing.value = true
    try {
      const ans = await submitAnswer({
        cards: deps.cardsClient,
        reportError: deps.reportError,
        vault: deps.vault.value,
        run: deps.run.value,
        card: one,
        how,
        elapsedMs,
      })
      if (!ans) return
      deps.cards.answers.value.push(ans)
    } finally {
      deps.writing.value = false
    }
    deps.cards.cardIndex.value += 1
    deps.cards.isAnswerShown.value = false
    deps.setPut(deps.now())
  }

  const takeBack = async () => {
    const last = deps.cards.answers.value[deps.cards.answers.value.length - 1]
    if (last === undefined || deps.writing.value) return
    deps.writing.value = true
    try {
      const ok = await revertAnswer({
        cards: deps.cardsClient,
        reportError: deps.reportError,
        vault: deps.vault.value,
        run: deps.run.value,
        answer: last,
      })
      if (!ok) return
    } finally {
      deps.writing.value = false
    }
    deps.cards.answers.value.pop()
    deps.cards.cardIndex.value = Math.max(0, deps.cards.cardIndex.value - 1)
    deps.cards.isAnswerShown.value = true
    deps.setPut(deps.now())
  }

  return { answer, takeBack }
}

function createSessionOpener(deps: {
  cardsClient: SessionClient
  reportError: (why: unknown) => void
  now: () => number
  cards: ReturnType<typeof useSessionCards>
  vault: Ref<string>
  run: Ref<string>
  setPut: (time: number) => void
}) {
  const start = async (
    id: string,
    deck: string,
    preset?: string | undefined,
  ): Promise<Report | null> => {
    const startedSession = await openSession({
      cards: deps.cardsClient,
      reportError: deps.reportError,
      id,
      deck,
      ...(preset !== undefined ? { preset } : {}),
    })
    if (!startedSession) return null
    deps.vault.value = id
    deps.run.value = startedSession.run
    deps.cards.cardList.value = startedSession.cards
    deps.cards.cardIndex.value = 0
    deps.cards.isAnswerShown.value = false
    deps.cards.answers.value = []
    deps.setPut(deps.now())
    return startedSession.report
  }

  const forget = () => {
    deps.cards.reset()
    deps.run.value = ''
    deps.vault.value = ''
  }

  return { start, forget }
}

export function useReviewSession(deps: SessionDeps) {
  const now = deps.now ?? (() => getNow())
  const vault = ref('')
  const run = ref('')
  const writing = ref(false)
  let displayedAt = now()
  const cards = useSessionCards()

  const { start, forget } = createSessionOpener({
    cardsClient: deps.cards,
    reportError: deps.reportError,
    now,
    cards,
    vault,
    run,
    setPut: (time) => {
      displayedAt = time
    },
  })

  const { answer, takeBack } = createAnswerActions({
    cardsClient: deps.cards,
    reportError: deps.reportError,
    now,
    vault,
    run,
    cards,
    writing,
    getPut: () => displayedAt,
    setPut: (time) => {
      displayedAt = time
    },
  })

  const api = {
    vault: vault as Ref<string>,
    run,
    cardList: cards.cardList,
    cardIndex: cards.cardIndex,
    isAnswerShown: cards.isAnswerShown,
    shown: cards.isAnswerShown,
    answers: cards.answers,
    writing,
    card: cards.card,
    left: cards.left,
    isOver: cards.isOver,
    over: cards.isOver,
    completedCount: cards.completedCount,
    done: cards.completedCount,
    start,
    show: () => {
      cards.isAnswerShown.value = true
    },
    answer,
    takeBack,
    forget,
  }
  return api
}

/** One card as the window holds it: the seconds come across as numbers. */
const createReviewCard = (one: SessionStart['cards'][number]): ReviewCard => ({
  deck: one.deck,
  section: one.section,
  cardId: one.cardId,
  template: one.template,
  heading: one.heading,
  front: one.front,
  back: one.back,
  isNew: one.isNew,
  ahead: one.intervals
    ? ({
        again: one.intervals.again,
        hard: one.intervals.hard,
        good: one.intervals.good,
        easy: one.intervals.easy,
      } satisfies Intervals)
    : null,
})
