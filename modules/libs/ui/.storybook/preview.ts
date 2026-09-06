import type { Preview } from '@storybook/vue3-vite'
import { configure } from 'storybook/test'
import { walk } from './keyboard'
import { faults } from './reach'
import '../src/tokens/theme.css'
import './preview.css'

// A story waits on a real browser drawing a frame, and two engines draw at
// once on one machine.
configure({ asyncUtilTimeout: 5_000 })

/**
 * The two multipliers, as far as each goes. The window refuses a number
 * outside these, so nothing reachable from the toolbar is a size a person
 * could not ask for.
 */
/** The story the keyboard walk is proved against, and what it must find there. */
const PROOF = { story: 'workspace--crowded', stops: 5 }

const INTERFACE = [0.8, 1, 1.25, 1.5, 2]
const READING = [0.8, 1, 1.25, 1.5, 1.75]

/** A multiplier as a toolbar item. */
const sizes = (among: readonly number[]) =>
  among.map((size) => ({ value: String(size), title: `${size}×` }))

/** The multiplier a toolbar is standing on. Anything else is as designed. */
const chosen = (held: unknown, among: readonly number[]) =>
  among.includes(Number(held)) ? String(Number(held)) : '1'

const preview: Preview = {
  parameters: {
    layout: 'fullscreen',
    controls: { matchers: { color: /(background|colou?r)$/i } },
    backgrounds: { disable: true },
    // A stencil cuts the cards a deck holds, and a card is what the two of
    // them come to, so the three read in that order.
    options: {
      storySort: { order: ['*', 'Flash Cards', ['Stencil', 'Deck', 'Card']] },
    },
  },
  globalTypes: {
    theme: {
      description: 'Light or dark, as the viewer would see it',
      toolbar: {
        title: 'Theme',
        icon: 'circlehollow',
        items: [
          { value: 'light', icon: 'sun', title: 'Light' },
          { value: 'dark', icon: 'moon', title: 'Dark' },
        ],
        dynamicTitle: true,
      },
    },
    interface: {
      description: 'How large the interface is drawn: chrome, controls, spacing, and the type in them',
      toolbar: {
        title: 'Interface',
        icon: 'grow',
        items: sizes(INTERFACE),
        dynamicTitle: true,
      },
    },
    font: {
      description: 'How large the text a person reads is set: a note, a book, an answer, the editor',
      toolbar: {
        title: 'Text',
        icon: 'paragraph',
        items: sizes(READING),
        dynamicTitle: true,
      },
    },
  },
  initialGlobals: { theme: 'light', interface: '1', font: '1' },
  afterEach: async (context) => {
    if (context.parameters['reach'] === false) return

    const found = await walk()

    // A walk that finds nothing passes everything. This one story is crowded
    // enough that finding nothing in it means the walk itself has stopped
    // working, and every other story's silence means nothing.
    if (context.id === PROOF.story && found.stops.length < PROOF.stops)
      throw new Error(
        `the keyboard walk found ${found.stops.length} stops in ${PROOF.story}, where it must find ${PROOF.stops}`,
      )

    const wrong = faults(found)
    if (wrong.length) throw new Error(`${context.id}: ${wrong.join('; ')}`)
  },
  decorators: [
    (story, context) => {
      const theme = context.globals['theme'] === 'dark' ? 'dark' : 'light'
      const root = document.documentElement
      root.style.colorScheme = theme
      // On the root, where the window puts them, so every story is looked at
      // through the same two numbers the window is drawn through.
      root.style.setProperty(
        '--numen-interface-scale',
        chosen(context.globals['interface'], INTERFACE),
      )
      root.style.setProperty('--numen-text-scale', chosen(context.globals['font'], READING))
      return { components: { story }, template: '<story />' }
    },
  ],
}

export default preview
