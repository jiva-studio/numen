/**
 * The flashcards window at rest: the decks a vault owes today, and one card
 * being answered.
 *
 * Nothing here waits on a vault. It is the screen a picture of the window is
 * taken from, so every piece is drawn in the state it settles in.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { dayAfter, getDayName, type HeatmapTally } from '@numen/ui'
import { h, type VNode } from 'vue'
import { Decks } from '@/pages/decks'
import { Session } from '@/pages/session'
import type { Preset, Settings } from '@/pages/decks'
import type { CardFace } from '@/entities/card'
import type { BudgetKeys, DeckCardsDue, VaultCardsDue } from '@/entities/vault'

const dayBefore = (day: string, back: number): string => dayAfter(day, -back)

/**
 * The day this window is read on. The grid of weeks runs up to the week it is
 * in, so the day the picture is taken is the day it is taken of.
 */
const TODAY = getDayName(new Date())

/** One day's answers, as the grid above the decks counts them. */
const tally = (answers: number, recalls: number): HeatmapTally => ({
  answered: answers,
  again: Math.round(answers * 0.12),
  hard: Math.round(answers * 0.18),
  good: Math.round(answers * 0.52),
  easy:
    answers - Math.round(answers * 0.12) - Math.round(answers * 0.18) - Math.round(answers * 0.52),
  asked: Math.round(answers * 0.8),
  recalled: recalls,
})

/**
 * A number between nought and one from a day's place and a name, so a year of
 * work is scattered the way a year is rather than stepping with the arithmetic
 * that made it.
 */
const scatter = (at: number, salt: number): number => {
  let x = Math.imul(at + salt * 7919 + 1, 2654435761)
  x ^= x >>> 15
  x = Math.imul(x, 2246822519)
  x ^= x >>> 13
  return ((x >>> 0) % 100_000) / 100_000
}

/**
 * A year of answering: worked most days, with a fortnight away in it, so the
 * grid is read as a person's year and not as a pattern.
 */
const DAYS = new Map<string, HeatmapTally>(
  Array.from({ length: 371 }, (_, back) => back)
    .filter((back) => scatter(back, 1) > 0.22 && !(back > 40 && back < 54))
    .map((back) => {
      // A quiet day and a long one, and the middle far commoner than either.
      const how = scatter(back, 2)
      const answered = Math.round(14 + how * how * 150)
      return [dayBefore(TODAY, back), tally(answered, Math.round(answered * 0.86))]
    }),
)

/** What falls on each day still to come, which is what the grid draws ahead. */
const DUE = new Map<string, number>(
  Array.from({ length: 34 }, (_, ahead) => [
    dayBefore(TODAY, -ahead - 1),
    Math.round(18 + scatter(ahead, 3) * 92),
  ]),
)

/** The budget that closes a day, in the words a preset writes the key in. */
const BY_MINUTES: BudgetKeys = { new: '', reviews: '', minutes: 'minutes_a_day' }
const BY_CARDS: BudgetKeys = { new: 'new_a_day', reviews: 'reviews_a_day', minutes: '' }

const settings = (over: Partial<Settings> = {}): Settings => ({
  goal: 'minutes',
  byDate: '',
  minutesADay: 25,
  newADay: 12,
  reviewsADay: 140,
  retention: 0.9,
  load: {},
  evenLoad: true,
  ...over,
})

const preset = (over: Partial<Preset> = {}): Preset => ({
  path: '',
  name: '',
  settings: settings(),
  decks: [],
  named: 0,
  faces: 0,
  cards: 0,
  budget: { new: 12, reviews: 140, minutes: 25 },
  closes: BY_MINUTES,
  answered: 0,
  answeredNew: 0,
  answeredReviews: 0,
  took: 0,
  paused: '',
  wrong: '',
  ...over,
})

const SANSKRIT = preset({
  path: 'Sanskrit.md',
  name: 'Sanskrit',
  decks: [
    'Sanskrit/Roots.md',
    'Sanskrit/Declensions.md',
    'Sanskrit/Sandhi.md',
    'Sanskrit/Compounds.md',
    'Sanskrit/Metre.md',
    'Sanskrit/Verbs.md',
    'Sanskrit/Vocabulary.md',
  ],
  named: 7,
  faces: 2_174,
  cards: 61,
  answered: 79,
  answeredNew: 11,
  answeredReviews: 68,
  took: 14,
})

const ANATOMY = preset({
  path: 'Anatomy.md',
  name: 'Anatomy',
  settings: settings({ goal: 'date', byDate: dayBefore(TODAY, -76) }),
  decks: [
    'Anatomy/Bones.md',
    'Anatomy/Muscles.md',
    'Anatomy/Nerves.md',
    'Anatomy/Vessels.md',
    'Anatomy/Joints.md',
    'Anatomy/Organs.md',
  ],
  named: 6,
  faces: 1_304,
  cards: 22,
  budget: { new: 20, reviews: 90, minutes: 0 },
  closes: BY_CARDS,
  answered: 96,
  answeredNew: 20,
  answeredReviews: 76,
  took: 21,
})

const PRESETS: readonly Preset[] = [SANSKRIT, ANATOMY]

const deck = (path: string, over: Partial<DeckCardsDue> = {}): DeckCardsDue => ({
  deck: path,
  faces: 400,
  due: 0,
  new: 0,
  learned: 0,
  unbegun: 0,
  ...over,
})

const DECKS: readonly DeckCardsDue[] = [
  deck('Sanskrit/Roots.md', { faces: 520, due: 34, new: 6, learned: 361 }),
  deck('Sanskrit/Declensions.md', { faces: 448, due: 21, new: 0, learned: 302 }),
  deck('Sanskrit/Sandhi.md', { faces: 272, due: 0, new: 0, learned: 249 }),
  deck('Anatomy/Bones.md', { faces: 336, due: 14, new: 8, learned: 188 }),
  deck('Anatomy/Muscles.md', { faces: 268, due: 0, new: 0, learned: 141, unbegun: 12 }),
  deck('Anatomy/Nerves.md', { faces: 214, due: 19, new: 4, learned: 96 }),
  deck('Sanskrit/Compounds.md', { faces: 186, due: 8, new: 0, learned: 121 }),
  deck('Sanskrit/Metre.md', { faces: 96, due: 0, new: 6, learned: 41 }),
  deck('Anatomy/Vessels.md', { faces: 158, due: 11, new: 0, learned: 83 }),
  deck('Sanskrit/Verbs.md', { faces: 240, due: 18, new: 0, learned: 152 }),
  deck('Sanskrit/Vocabulary.md', { faces: 412, due: 27, new: 0, learned: 268 }),
  deck('Anatomy/Joints.md', { faces: 132, due: 9, new: 0, learned: 71 }),
  deck('Anatomy/Organs.md', { faces: 196, due: 0, new: 5, learned: 88 }),
]

const VAULT: VaultCardsDue = {
  vault: 'v1',
  name: 'Studies',
  path: '/home/you/Studies',
  counted: true,
  faces: 3_478,
  due: 161,
  new: 29,
  decks: DECKS,
  presets: [],
  reading: false,
  unread: '',
}

/** The preset each deck is scheduled by, which is what the list says under a name. */
const BY_DECK = new Map<string, Preset>(
  PRESETS.flatMap((one) => one.decks.map((path) => [path, one] as const)),
)

/** One card face, as the session puts it. */
const CARD: CardFace = {
  deck: 'Sanskrit/Roots.md',
  section: 'Verbs of going',
  mark: 'k7m2xq9fzp',
  face: 'Recognise',
  heading: 'gam',
  front:
    '<p><strong>gam</strong> — गम्</p>\n<p>Which class does it take, and what is its present stem?</p>',
  back:
    '<p><strong>Meaning:</strong> to go, to move</p>\n' +
    '<p><strong>Class:</strong> 1, thematic</p>\n' +
    '<p><strong>Present:</strong> <em>gacchati</em>, from the reduplicated stem</p>\n' +
    '<p><strong>Perfect:</strong> <em>jagāma</em></p>\n' +
    '<p><strong>Aorist:</strong> <em>agamat</em></p>\n' +
    '<p>The root is written up in <a href="Sanskrit/Roots.md">Roots</a>, ' +
    'beside the other verbs of going.</p>',
  seen: true,
  ahead: { again: 60, hard: 4 * 86_400, good: 11 * 86_400, easy: 26 * 86_400 },
}

/** The window the two screens are drawn in, as the application draws them. */
const frame = (screen: VNode) => ({
  setup: () => () =>
    h(
      'main',
      {
        style: {
          display: 'flex',
          flexDirection: 'column',
          blockSize: '100vh',
          padding: 'var(--numen-inset-wide)',
          gap: 'var(--numen-inset)',
          fontFamily: 'var(--numen-font-sans)',
          fontSize: 'var(--numen-font-size)',
          lineHeight: 'var(--numen-line-height)',
        },
      },
      [screen],
    ),
})

const meta: Meta = {
  title: 'Flash Cards/Window',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

/** What the vault owes today: the goals, the days answered on, and the decks. */
export const CardsDue: Story = {
  render: () =>
    frame(
      h(Decks, {
        vault: VAULT,
        days: DAYS,
        due: DUE,
        presets: PRESETS,
        byDeck: BY_DECK,
        scheduled: true,
        today: TODAY,
      }),
    ),
}

/** One card, answered: the four ways of saying it, and where each leaves it. */
export const Reviewing: Story = {
  render: () =>
    frame(
      h(Session, {
        card: CARD,
        shown: true,
        left: 161,
        takenBack: true,
        at: 'here',
      }),
    ),
}
