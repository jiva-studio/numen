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

import { RATING_BY_GRADE } from '../../lib/card/grades'
import type { Rating, UndoReviewResponse } from '../../../lib/wire/review'
import type { ReviewCard, Intervals } from '../../lib/card/cards'
import type { Grade } from '../../lib/card/grades'
import type { Caught } from '@numen/plugin-sdk'

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
  undoReview(params: { vault: string; run: string; answer: string }): Promise<UndoReviewResponse>
}

/** What opening a session returns. */
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
  readonly unwritten: readonly string[]
  readonly skipped: number
}

/** What a session is built over: the application client, and the error reporter. */
export interface SessionDeps {
  cards: SessionClient
  reportError(error: Caught): void
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

  return {
    queue: { cardList: request, cardIndex, answers, card, isAnswerShown },
    progress: { left: remaining, isOver, completedCount: isDone },
    reset,
  }
}

async function openSession(deps: {
  cards: SessionClient
  reportError: (error: Caught) => void
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
  reportError: (error: Caught) => void
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
      cardId: deps.card.address.cardId,
      template: deps.card.address.template,
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
  reportError: (error: Caught) => void
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
  reportError: (error: Caught) => void
  now: () => number
  vault: Ref<string>
  run: Ref<string>
  cards: ReturnType<typeof useSessionCards>
  writing: Ref<boolean>
  put: { get: () => number; set: (time: number) => void }
}) {
  const answer = async (how: Grade) => {
    const one = deps.cards.queue.card.value
    if (!one || !deps.cards.queue.isAnswerShown.value || deps.writing.value) return
    const elapsedMs = deps.now() - deps.put.get()
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
      deps.cards.queue.answers.value.push(ans)
    } finally {
      deps.writing.value = false
    }
    deps.cards.queue.cardIndex.value += 1
    deps.cards.queue.isAnswerShown.value = false
    deps.put.set(deps.now())
  }

  const takeBack = async () => {
    const last = deps.cards.queue.answers.value[deps.cards.queue.answers.value.length - 1]
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
    deps.cards.queue.answers.value.pop()
    deps.cards.queue.cardIndex.value = Math.max(0, deps.cards.queue.cardIndex.value - 1)
    deps.cards.queue.isAnswerShown.value = true
    deps.put.set(deps.now())
  }

  return { answer, takeBack }
}

function createSessionOpener(deps: {
  cardsClient: SessionClient
  reportError: (error: Caught) => void
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
    deps.cards.queue.cardList.value = startedSession.cards
    deps.cards.queue.cardIndex.value = 0
    deps.cards.queue.isAnswerShown.value = false
    deps.cards.queue.answers.value = []
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

  const opener = createSessionOpener({
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

  const answering = createAnswerActions({
    cardsClient: deps.cards,
    reportError: deps.reportError,
    now,
    vault,
    run,
    cards,
    writing,
    put: {
      get: () => displayedAt,
      set: (time) => {
        displayedAt = time
      },
    },
  })

  return {
    ids: { vault: vault as Ref<string>, run },
    writing,
    cards,
    opener,
    answering,
    show: () => {
      cards.queue.isAnswerShown.value = true
    },
  }
}

const createReviewCard = (one: SessionStart['cards'][number]): ReviewCard => ({
  address: { deck: one.deck, section: one.section, cardId: one.cardId, template: one.template },
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
