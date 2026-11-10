/**
 * The card model tab: the fields a card model names, and the templates that show them.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { fn } from 'storybook/test'
import { computed } from 'vue'
import { createMarks, type Marks } from '../../../lib/marks'
import CardModelTab from './CardModelTab.vue'
import type { CardModelTabState } from '../../model/card-model/types'

const meta: Meta = {
  title: 'Flashcards/Card model',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

const NO_MARKS: Marks = createMarks([], [], [])

const WORD = { name: 'Word', fields: ['Word', 'Reading', 'Meaning', 'In a sentence'] }

const CARD_MODEL_STATE: CardModelTabState = {
  id: 'Sanskrit/Word.md',
  data: {
    note: computed(() => ({
      path: 'Sanskrit/Word.md',
      body: '',
      state: 'clean',
      error: null,
      isRead: true,
    })),
    cardModel: computed(() => ({
      fields: WORD.fields,
      preamble: '',
      templates: [
        {
          id: 'recognise',
          name: 'Recognise',
          preamble: '',
          front: '<p>{{Word}} — <i>{{Reading}}</i></p>',
          back: '<p><b>Meaning:</b> {{Meaning}}</p>\n<p><b>In a sentence:</b> {{In a sentence}}</p>',
        },
        {
          id: 'produce',
          name: 'Produce',
          preamble: '',
          front: '<p>Which word means <i>{{Meaning}}</i>?</p>',
          back: '<p>{{Word}} — <i>{{Reading}}</i></p>\n<p>{{In a sentence}}</p>',
        },
        {
          id: 'read-it',
          name: 'Read it',
          preamble: '',
          front: '<p><i>{{Reading}}</i></p>',
          back: '<p>{{Word}}</p>\n<p><b>Meaning:</b> {{Meaning}}</p>',
        },
      ],
      tail: '',
    })),
    marks: computed(() => NO_MARKS),
    errorMessage: computed(() => ''),
  },
  fields: {
    addField: fn(),
    renameField: fn(),
    removeField: fn(),
    moveField: fn(),
  },
  templates: {
    addTemplate: fn(),
    renameTemplate: fn(),
    removeTemplate: fn(),
    moveTemplate: fn(),
    writeTemplateHalf: fn(),
  },
  file: {
    keepMine: fn(),
    takeFile: fn(),
    close: fn(),
  },
}

/** The fields a card model names, and the templates that show them. */
export const CardModel: Story = {
  render: () => ({
    components: { CardModelTab },
    setup: () => ({ state: CARD_MODEL_STATE }),
    template: '<div style="height: 100vh"><CardModelTab :state="state" /></div>',
  }),
}
