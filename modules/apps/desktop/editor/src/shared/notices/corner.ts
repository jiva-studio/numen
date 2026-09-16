/**
 * What the corner of the window draws.
 *
 * Everything the window has to say is one card: work it is doing, a state it is
 * in, and what it answered the last thing it was asked. A new kind of any of
 * them is an entry in one of the three lists and nothing here.
 */
import { createNotice } from '@numen/ui'
import type { Notice, Stay, Tone } from '@numen/ui'
import type { Task } from './task'
import { wordsOnly, type IndexCoverage } from './coverage'
import type { MessageKind, WindowMessage } from './messages'

/** The sentences the corner draws that are the window's own. */
export interface Words {
  /** The vault is there, and the window is not being told when it changes. */
  readonly unwatched: string
  /** The vault itself could not be read. */
  readonly unread: string
  /** The vault is still being read for the first time. */
  readonly reading: string
  /** The vault could not be read, so there is nothing on screen. */
  readonly nothingRead: string
  /** One way of asking is missing and nothing is going to bring it. */
  readonly wordsOnly: string
}

/** What is so about the window, whatever it is doing. */
export interface State {
  /** The vault is there, and the window is not being told when it changes. */
  readonly unwatched: string
  /** The vault itself could not be read. */
  readonly unread: string
  /** What the window lost touch with. */
  readonly lost: string
  /** The vault is still being read for the first time. */
  readonly reading: boolean
  /** Whether the vault holds a note to show at all. */
  readonly hasNote: boolean
}

/** How each kind of word is drawn, and how long it stands. */
const manner: Record<MessageKind, { tone: Tone; stay: Stay }> = {
  error: { tone: 'alarm', stay: 'kept' },
  caution: { tone: 'caution', stay: 'kept' },
  report: { tone: 'plain', stay: 'read' },
  state: { tone: 'plain', stay: 'holds' },
}

/** One state of the window, where it is in that state. */
const soThat = (
  id: string,
  text: string,
  how: { about?: string; tone?: Tone; isAsked?: boolean } = {},
): readonly Notice[] =>
  text === ''
    ? []
    : [
        {
          id,
          text,
          about: how.about ?? '',
          tone: how.tone ?? 'plain',
          working: false,
          isAsked: how.isAsked ?? true,
          stay: 'holds',
        },
      ]

/**
 * Whether one reason is the other with what a pass was doing put in front of
 * it. That is how a reason arrives from the core: `doing this: what went
 * wrong`.
 */
const isSameReason = (outer: string, inner: string): boolean =>
  outer === inner || outer.endsWith(`: ${inner}`)

/**
 * The work worth a card of its own.
 *
 * One thing goes wrong and every pass waiting on it stops with the same
 * sentence. The one that says it and nothing more is the card, and of two
 * saying the same thing it is the one that said it first.
 */
const alone = (tasks: readonly Task[]): readonly Task[] =>
  tasks.filter((at, index) => {
    if (at.error === '') return true
    return !tasks.some(
      (other, was) =>
        other.error !== '' &&
        other !== at &&
        isSameReason(at.error, other.error) &&
        (other.error.length < at.error.length || was < index),
    )
  })

/**
 * The cards the corner draws.
 *
 * Work first, because only work has numbers that move, and a state or a word
 * arriving must not shift what a person is reading. Within each of the three
 * the order is the order it was first seen in.
 */
export const cornerOf = (
  tasks: readonly Task[],
  messages: readonly WindowMessage[],
  state: State,
  vault: IndexCoverage,
  words: Words,
): readonly Notice[] => {
  const working: readonly Notice[] = alone(tasks).map(createNotice)

  const so: Notice[] = [
    ...soThat('unwatched', state.unwatched && words.unwatched, {
      about: state.unwatched,
      tone: 'caution',
    }),
    ...soThat('unread', state.unread && words.unread, {
      about: state.unread,
      tone: 'caution',
    }),
    ...soThat('lost', state.lost, { tone: 'caution' }),
    ...soThat('reading', state.reading ? words.reading : ''),
    // One at a time: a vault still being read has not finished reading nothing.
    ...soThat(
      'nothingRead',
      !state.reading && !state.hasNote && state.unread ? words.nothingRead : '',
    ),
    // Said once and quietly, and it is so whether or not anything is running.
    ...soThat('wordsOnly', wordsOnly(vault) ? words.wordsOnly : '', { isAsked: false }),
  ]

  const said: Notice[] = messages.map((one) => ({
    id: one.id,
    text: one.text,
    working: false,
    isAsked: true,
    ...manner[one.kind],
  }))

  return [...working, ...so, ...said]
}
