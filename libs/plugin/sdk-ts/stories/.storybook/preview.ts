import type { Preview } from '@storybook/vue3-vite'
import { configure } from 'storybook/test'
// The one keyboard walk and the one toolbar, where they are written. A plugin
// draws the components, so it is judged by what they are judged by.
import { reachCheck, type Proof } from '../../../../ui/components/.storybook/check'
import { toolbar } from '../../../../ui/components/.storybook/toolbar'
import '@numen/ui/styles.css'
import 'virtual:plugin-utilities.css'
import './preview.css'

// A story waits on a real browser drawing a frame, and two engines draw at
// once on one machine.
configure({ asyncUtilTimeout: 5_000 })

/**
 * The story the plugins' walk is proved against. The number is the fewest stops
 * of its tab order in either engine: WebKit on macOS does not stop on buttons,
 * so WebKit finds six stops there where Chromium finds seven.
 */
const PROOF: Proof = { story: 'flash-cards-card--what-is-wrong-with-it', stops: 6 }

const preview: Preview = {
  parameters: {
    layout: 'fullscreen',
    backgrounds: { disable: true },
    // A card model cuts the cards a deck holds, and a card is what the two of
    // them come to, so the three read in that order.
    options: {
      storySort: { order: ['*', 'Flash Cards', ['Card model', 'Deck', 'Card']] },
    },
  },
  ...toolbar,
  afterEach: reachCheck(PROOF),
}

export default preview
