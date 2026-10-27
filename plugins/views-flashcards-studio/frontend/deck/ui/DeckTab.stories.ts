/**
 * The deck tab: the cards of a deck under the sections they stand in, the card models that cut them and
 * the schedule it keeps.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { fn } from 'storybook/test'
import { computed } from 'vue'
import { createMarks, type Marks } from '../../cards'
import type { DeckTabState } from '../types'
import DeckTab from './DeckTab.vue'

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

const CARDS = [
  card('gam', 'going', 'Root', [
    { field: 'Root', text: 'gam' },
    { field: 'Class', text: '1, thematic' },
    { field: 'Present', text: 'gacchati' },
    { field: 'Meaning', text: 'to go, to move' },
  ]),
  card('car', 'going', 'Root', [
    { field: 'Root', text: 'car' },
    { field: 'Class', text: '1' },
    { field: 'Present', text: 'carati' },
    { field: 'Meaning', text: 'to walk, to wander' },
  ]),
  card('sthā', 'going', 'Root', [
    { field: 'Root', text: 'sthā' },
    { field: 'Class', text: '1' },
    { field: 'Present', text: 'tiṣṭhati' },
    { field: 'Meaning', text: 'to stand, to stay' },
  ]),
  card('i', 'going', 'Root', [
    { field: 'Root', text: 'i' },
    { field: 'Class', text: '2' },
    { field: 'Present', text: 'eti' },
    { field: 'Meaning', text: 'to go' },
  ]),
  card('vid', 'knowing', 'Root', [
    { field: 'Root', text: 'vid' },
    { field: 'Class', text: '2' },
    { field: 'Present', text: 'vetti' },
    { field: 'Meaning', text: 'to know' },
  ]),
  card('jñā', 'knowing', 'Root', [
    { field: 'Root', text: 'jñā' },
    { field: 'Class', text: '9' },
    { field: 'Present', text: 'jānāti' },
    { field: 'Meaning', text: 'to know, to recognise' },
  ]),
  card('smṛ', 'knowing', 'Root', [
    { field: 'Root', text: 'smṛ' },
    { field: 'Class', text: '1' },
    { field: 'Present', text: 'smarati' },
    { field: 'Meaning', text: 'to remember' },
  ]),
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
  note: computed(() => ({ path: 'Sanskrit/Roots.md', body: '', state: 'clean', error: null })),
  deck: computed(() => ({ preamble: '', cards: [], sections: [], tail: '' })),
  deckCards: computed(() => CARDS),
  sections: computed(() => SECTIONS),
  cardModels: computed(() => [ROOT_CUT, WORD]),
  marks: computed(() => NO_MARKS),
  errorMessage: computed(() => ''),
  scheduled: computed(() => ({ path: 'Sanskrit.md', name: 'Sanskrit', errorMessage: '' })),
  choices: computed(() => [
    { path: '', name: 'Defaults' },
    { path: 'Sanskrit.md', name: 'Sanskrit' },
    { path: 'Anatomy.md', name: 'Anatomy' },
  ]),
  setSchedule: fn(),
  addCard: fn(),
  removeCard: fn(),
  moveCard: fn(),
  writeCardField: fn(),
  addSection: fn(),
  renameSection: fn(),
  removeSection: fn(),
  keepMine: fn(),
  takeFile: fn(),
  close: fn(),
}

/** The cards of a deck, under the sections they stand in. */
export const Deck: Story = {
  render: () => ({
    components: { DeckTab },
    setup: () => ({ state: DECK_STATE }),
    template: '<div style="height: 100vh"><DeckTab :state="state" /></div>',
  }),
}
