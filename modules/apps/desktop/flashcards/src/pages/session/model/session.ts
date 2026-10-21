/**
 * One session: the cards left to ask, where in them a person is, and what has
 * been written into the vault.
 *
 * Apart from the template because these are the rules that decide what a person
 * is asked and what their vault is told, and a rule inside a component is a rule
 * only exercised by looking at the screen.
 */
import { computed, ref, shallowRef } from 'vue'
import type { Ref } from 'vue'

import { RATING_BY_GRADE } from '@/entities/card'
import type { CardFace, Grade, Intervals } from '@/entities/card'

/** What a session asks of the application, and no more of it than that. */
export interface SessionClient {
  /**
   * Deck is one deck, or empty for every deck the vault holds. Preset holds it
   * to the decks one preset schedules, and to the budget that preset keeps.
   */
  startSession(params: { vault: string; deck: string; preset?: string }): Promise<SessionStart>
  answerCard(params: {
    vault: string
    run: string
    mark: string
    face: string
    rating: number
    tookMs: bigint
  }): Promise<{ answer: string }>
  takeBackAnswer(params: { vault: string; run: string; answer: string }): Promise<unknown>
}

/** What opening a session comes back with. */
export interface SessionStart {
  run: string
  cards: readonly {
    deck: string
    section: string
    mark: string
    face: string
    heading: string
    front: string
    back: string
    isNew: boolean
    intervals?: { again: bigint; hard: bigint; good: bigint; easy: bigint } | undefined
  }[]
  unwritten: readonly string[]
  skipped: number
}

/** What a session could not act on, for the window to say once as it opens. */
export interface Report {
  /** The decks holding a card that could not be given a mark. */
  readonly unwritten: readonly string[]
  /** How many lines of the vault's answers could not be read. */
  readonly skipped: number
}

/** What a session is built over: the application, and what it says went wrong. */
export interface SessionDeps {
  cards: SessionClient
  reportError(why: unknown): void
  /** When it is, in milliseconds. How long a card stood there is measured with it. */
  now?(): number
}

function useSessionCards() {
  const asked = shallowRef<readonly CardFace[]>([])
  const at = ref(0)
  const shown = ref(false)
  const answers = ref<string[]>([])

  const card = computed<CardFace | null>(() => asked.value[at.value] ?? null)
  const left = computed(() => asked.value.length - at.value)
  const isOver = computed(() => card.value === null)
  const done = computed(() => answers.value.length)

  const reset = () => {
    asked.value = []
    at.value = 0
    shown.value = false
    answers.value = []
  }

  return { asked, at, shown, answers, card, left, over, done, reset }
}

async function openSession(deps: {
  cards: SessionClient
  reportError: (why: unknown) => void
  id: string
  deck: string
  preset?: string
}): Promise<{ run: string; cards: readonly CardFace[]; report: Report } | null> {
  try {
    const opened = await deps.cards.startSession(
      deps.preset === undefined
        ? { vault: deps.id, deck: deps.deck }
        : { vault: deps.id, deck: deps.deck, preset: deps.preset },
    )
    return {
      run: opened.run,
      cards: opened.cards.map(createCardFace),
      report: { unwritten: opened.unwritten, skipped: opened.skipped },
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
  card: CardFace
  how: Grade
  elapsedMs: number
}): Promise<string | null> {
  try {
    const response = await deps.cards.answerCard({
      vault: deps.vault,
      run: deps.run,
      mark: deps.card.mark,
      face: deps.card.face,
      rating: RATING_BY_GRADE[deps.how],
      tookMs: BigInt(deps.elapsedMs),
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
    await deps.cards.takeBackAnswer({ vault: deps.vault, run: deps.run, answer: deps.answer })
    return true
  } catch (why) {
    deps.reportError(why)
    return false
  }
}

function useAnswerActions(deps: {
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
    if (!one || !deps.cards.shown.value || deps.writing.value) return
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
    deps.cards.at.value += 1
    deps.cards.shown.value = false
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
    deps.cards.at.value = Math.max(0, deps.cards.at.value - 1)
    deps.cards.shown.value = true
    deps.setPut(deps.now())
  }

  return { answer, takeBack }
}

function useSessionOpener(deps: {
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
    const opened = await openSession({
      cards: deps.cardsClient,
      reportError: deps.reportError,
      id,
      deck,
      ...(preset !== undefined ? { preset } : {}),
    })
    if (!opened) return null
    deps.vault.value = id
    deps.run.value = opened.run
    deps.cards.asked.value = opened.cards
    deps.cards.at.value = 0
    deps.cards.shown.value = false
    deps.cards.answers.value = []
    deps.setPut(deps.now())
    return opened.report
  }

  const forget = () => {
    deps.cards.reset()
    deps.run.value = ''
    deps.vault.value = ''
  }

  return { start, forget }
}

export function useReviewSession(deps: SessionDeps) {
  const now = deps.now ?? (() => Date.now())
  const vault = ref('')
  const run = ref('')
  const writing = ref(false)
  let put = now()
  const cards = useSessionCards()

  const { start, forget } = useSessionOpener({
    cardsClient: deps.cards,
    reportError: deps.reportError,
    now,
    cards,
    vault,
    run,
    setPut: (time) => {
      put = time
    },
  })

  const { answer, takeBack } = useAnswerActions({
    cardsClient: deps.cards,
    reportError: deps.reportError,
    now,
    vault,
    run,
    cards,
    writing,
    getPut: () => put,
    setPut: (time) => {
      put = time
    },
  })

  return {
    vault: vault as Ref<string>,
    run,
    asked: cards.asked,
    at: cards.at,
    shown: cards.shown,
    answers: cards.answers,
    writing,
    card: cards.card,
    left: cards.left,
    over: cards.over,
    done: cards.done,
    start,
    show: () => {
      cards.shown.value = true
    },
    answer,
    takeBack,
    forget,
  }
}

/** One card as the window holds it: the seconds come across as numbers. */
const createCardFace = (one: SessionStart['cards'][number]): CardFace => ({
  deck: one.deck,
  section: one.section,
  mark: one.mark,
  face: one.face,
  heading: one.heading,
  front: one.front,
  back: one.back,
  isNew: one.isNew,
  ahead: one.intervals
    ? ({
        again: Number(one.intervals.again),
        hard: Number(one.intervals.hard),
        good: Number(one.intervals.good),
        easy: Number(one.intervals.easy),
      } satisfies Intervals)
    : null,
})
