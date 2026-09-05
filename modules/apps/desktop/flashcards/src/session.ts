/**
 * One sitting: the cards left to ask, where in them a person is, and what has
 * been written into the vault.
 *
 * Apart from the template because these are the rules that decide what a person
 * is asked and what their vault is told, and a rule inside a component is a rule
 * only exercised by looking at the screen.
 */
import { computed, ref } from 'vue'
import type { Ref } from 'vue'

import { rated } from './core'
import type { CardFace, Grade, Intervals } from './core'

/** What a sitting asks of the application, and no more of it than that. */
export interface SessionClient {
  /**
   * Deck is one deck, or empty for every deck the vault holds. Preset holds it
   * to the decks one preset schedules, and to the budget that preset keeps.
   */
  startSession(said: { vault: string; deck: string; preset?: string }): Promise<SessionStart>
  answerCard(said: {
    vault: string
    run: string
    card: string
    face: string
    rating: number
    tookMs: bigint
  }): Promise<{ answer: string }>
  takeBackAnswer(said: { vault: string; run: string; answer: string }): Promise<unknown>
}

/** What opening a sitting comes back with. */
export interface SessionStart {
  run: string
  asked: readonly {
    deck: string
    section: string
    card: string
    face: string
    heading: string
    front: string
    back: string
    seen: boolean
    ahead?: { again: bigint; hard: bigint; good: bigint; easy: bigint } | undefined
  }[]
  unwritten: readonly string[]
  skipped: number
}

/** What a sitting could not act on, for the window to say once as it opens. */
export interface Report {
  /** The decks holding a card that could not be given a mark. */
  readonly unwritten: readonly string[]
  /** How many lines of the vault's answers could not be read. */
  readonly skipped: number
}

/** What a sitting is built over: the application, and what it says went wrong. */
export interface SessionDeps {
  cards: SessionClient
  failed(why: unknown): void
  /** When it is, in milliseconds. How long a card stood there is measured with it. */
  now?(): number
}

export function session(deps: SessionDeps) {
  const now = deps.now ?? (() => Date.now())

  const vault = ref('')
  const run = ref('')
  const asked = ref<readonly CardFace[]>([])
  const at = ref(0)
  const shown = ref(false)
  const answers = ref<string[]>([])

  /**
   * Whether an answer is on its way to the vault. A card is answered once: the
   * line is written before the next card is put up, and a second press while
   * that is happening would write the same card twice and skip the one after
   * it.
   */
  const writing = ref(false)

  /** When the card now in front of the person was put there. */
  let put = now()

  const card = computed<CardFace | null>(() => asked.value[at.value] ?? null)
  const left = computed(() => asked.value.length - at.value)
  const over = computed(() => card.value === null)

  /** How many answers this sitting has written, which is what stands at the end. */
  const done = computed(() => answers.value.length)

  /**
   * The sitting let go of. What was answered is in the vault, and the next
   * sitting is opened whole: a card kept here is one a screen could fall back
   * to showing with nothing able to answer it.
   */
  const forget = () => {
    asked.value = []
    at.value = 0
    shown.value = false
    answers.value = []
    run.value = ''
  }

  /**
   * Sit down to a vault, or to one deck of it. What comes back is what the
   * sitting could not act on, and nothing when it could not be opened at all.
   */
  /**
   * Preset is the note one preset stands in, and the empty path is the preset
   * that schedules the decks naming none. Naming none at all sits to the deck.
   */
  const start = async (
    named: string,
    deck: string,
    preset?: string,
  ): Promise<Report | null> => {
    try {
      const opened = await deps.cards.startSession(
        preset === undefined
          ? { vault: named, deck }
          : { vault: named, deck, preset },
      )
      vault.value = named
      run.value = opened.run
      asked.value = opened.asked.map(asking)
      at.value = 0
      shown.value = false
      answers.value = []
      put = now()
      return { unwritten: opened.unwritten, skipped: opened.skipped }
    } catch (why) {
      deps.failed(why)
      return null
    }
  }

  const show = () => {
    shown.value = true
  }

  /**
   * Answer the card in front of the person. An answer that could not be written
   * leaves the card where it was: what a person said is theirs, and a card
   * moved past with nothing recorded is an answer lost.
   */
  const answer = async (how: Grade) => {
    const one = card.value
    if (!one || !shown.value || writing.value) return
    const took = now() - put
    writing.value = true
    try {
      const given = await deps.cards.answerCard({
        vault: vault.value,
        run: run.value,
        card: one.card,
        face: one.face,
        rating: rated[how],
        tookMs: BigInt(took),
      })
      answers.value.push(given.answer)
    } catch (why) {
      deps.failed(why)
      return
    } finally {
      writing.value = false
    }
    at.value += 1
    shown.value = false
    put = now()
  }

  /**
   * Take the last answer back. The card comes back with its answer showing,
   * which is where the person was when they pressed the wrong key.
   */
  const takeBack = async () => {
    const last = answers.value[answers.value.length - 1]
    if (last === undefined || writing.value) return
    writing.value = true
    try {
      await deps.cards.takeBackAnswer({ vault: vault.value, run: run.value, answer: last })
    } catch (why) {
      deps.failed(why)
      return
    } finally {
      writing.value = false
    }
    answers.value.pop()
    at.value = Math.max(0, at.value - 1)
    shown.value = true
    put = now()
  }

  return {
    vault: vault as Ref<string>,
    run,
    asked,
    at,
    shown,
    answers,
    writing,
    card,
    left,
    over,
    done,
    start,
    show,
    answer,
    takeBack,
    forget,
  }
}

/** One card as the window holds it: the seconds come across as numbers. */
const asking = (one: SessionStart['asked'][number]): CardFace => ({
  deck: one.deck,
  section: one.section,
  card: one.card,
  face: one.face,
  heading: one.heading,
  front: one.front,
  back: one.back,
  seen: one.seen,
  ahead: one.ahead
    ? ({
        again: Number(one.ahead.again),
        hard: Number(one.ahead.hard),
        good: Number(one.ahead.good),
        easy: Number(one.ahead.easy),
      } satisfies Intervals)
    : null,
})
