/**
 * The stencil tab: the fields a stencil names, and the faces that show them.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { fn } from 'storybook/test'
import { computed } from 'vue'
import { createMarks, type Marks } from '../../cards'
import type { StencilTabState } from '../types'
import StencilTab from './StencilTab.vue'

const meta: Meta = {
  title: 'Flashcards/Stencil',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

const NO_MARKS: Marks = createMarks([], [], [])

const WORD = { name: 'Word', fields: ['Word', 'Reading', 'Meaning', 'In a sentence'] }

const STENCIL_STATE: StencilTabState = {
  id: 'Sanskrit/Word.md',
  note: computed(() => ({ path: 'Sanskrit/Word.md', body: '', state: 'clean', error: null })),
  stencil: computed(() => ({
    fields: WORD.fields,
    preamble: '',
    faces: [
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
  addField: fn(),
  renameField: fn(),
  removeField: fn(),
  moveField: fn(),
  addFace: fn(),
  renameFace: fn(),
  removeFace: fn(),
  moveFace: fn(),
  writeFaceHalf: fn(),
  keepMine: fn(),
  takeFile: fn(),
  close: fn(),
}

/** The fields a stencil names, and the faces that show them. */
export const Stencil: Story = {
  render: () => ({
    components: { StencilTab },
    setup: () => ({ state: STENCIL_STATE }),
    template: '<div style="height: 100vh"><StencilTab :state="state" /></div>',
  }),
}
