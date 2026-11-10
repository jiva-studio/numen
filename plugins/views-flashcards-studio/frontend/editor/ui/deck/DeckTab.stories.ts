/**
 * The deck tab: the cards of a deck under the sections they stand in, the card models that cut them and
 * the schedule it keeps.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { fn } from 'storybook/test'
import { computed } from 'vue'
import { createMarks, type Marks } from '../../../lib/marks'
import DeckTab from './DeckTab.vue'
import type { DeckTabState } from '../../model/deck/tabState'

const meta: Meta = {
  title: 'Flashcards/Deck',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

const NO_MARKS: Marks = createMarks([], [], [])

const WORD = { name: 'Word', fields: ['Word', 'Reading', 'Meaning', 'In a sentence'] }
const ROOT_CUT = { name: 'Root', fields: ['Root', 'Class', 'Present', 'Meaning'] }

const card = (
  id: string,
  section: string,
  cardModelLink: string,
  values: readonly { field: string; text: string }[],
) => ({ id, section, cardModel: cardModelLink, filled: values })

const root = (
  id: string,
  section: string,
  [klass, present, meaning]: readonly [string, string, string],
) =>
  card(id, section, 'Root', [
    { field: 'Root', text: id },
    { field: 'Class', text: klass },
    { field: 'Present', text: present },
    { field: 'Meaning', text: meaning },
  ])

const CARDS = [
  root('gam', 'going', ['1, thematic', 'gacchati', 'to go, to move']),
  root('car', 'going', ['1', 'carati', 'to walk, to wander']),
  root('sthā', 'going', ['1', 'tiṣṭhati', 'to stand, to stay']),
  root('i', 'going', ['2', 'eti', 'to go']),
  root('vid', 'knowing', ['2', 'vetti', 'to know']),
  root('jñā', 'knowing', ['9', 'jānāti', 'to know, to recognise']),
  root('smṛ', 'knowing', ['1', 'smarati', 'to remember']),
  card('mārga', 'nouns', 'Word', [
    { field: 'Word', text: 'mārga' },
    { field: 'Reading', text: 'मार्ग' },
    { field: 'Meaning', text: 'a road, a way' },
    { field: 'In a sentence', text: 'sa mārgaṃ gacchati' },
  ]),
]

const SECTIONS = [
  { id: 'going', name: 'Verbs of going' },
  { id: 'knowing', name: 'Verbs of knowing' },
  { id: 'nouns', name: 'Nouns off them' },
]

const DECK_STATE: DeckTabState = {
  id: 'Sanskrit/Roots.md',
  data: {
    note: computed(() => ({
      path: 'Sanskrit/Roots.md',
      body: '',
      state: 'clean',
      error: null,
      isRead: true,
    })),
    deck: computed(() => ({ preamble: '', cards: [], sections: [], tail: '' })),
    deckCards: computed(() => CARDS),
    sections: computed(() => SECTIONS),
    cardModels: computed(() => [ROOT_CUT, WORD]),
    marks: computed(() => NO_MARKS),
    errorMessage: computed(() => ''),
  },
  schedule: {
    scheduled: computed(() => ({ path: 'Sanskrit.md', name: 'Sanskrit', errorMessage: '' })),
    choices: computed(() => [
      { path: '', name: 'Defaults' },
      { path: 'Sanskrit.md', name: 'Sanskrit' },
      { path: 'Anatomy.md', name: 'Anatomy' },
    ]),
    setSchedule: fn(),
  },
  actions: {
    addCard: fn(),
    removeCard: fn(),
    moveCard: fn(),
    writeCardField: fn(),
    addSection: fn(),
    renameSection: fn(),
    removeSection: fn(),
  },
  file: {
    keepMine: fn(),
    takeFile: fn(),
    close: fn(),
  },
}

/** The cards of a deck, under the sections they stand in. */
export const Deck: Story = {
  render: () => ({
    components: { DeckTab },
    setup: () => ({ state: DECK_STATE }),
    template: '<div style="height: 100vh"><DeckTab :state="state" /></div>',
  }),
}
