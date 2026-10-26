import type { Preview } from '@storybook/vue3-vite'
import { configure } from 'storybook/test'
// The one keyboard walk and the one toolbar, where they are written.
import { reachCheck, type Proof } from '../../core/.storybook/check'
import { toolbar } from '../../core/.storybook/toolbar'
import '@numen/ui/styles.css'
import '../src/styles.css'
import './preview.css'

// A story waits on a real browser drawing a frame, and two engines draw at
// once on one machine.
configure({ asyncUtilTimeout: 5_000 })

/**
 * The story the keyboard walk is proved against, and what it must find there.
 * The number is the whole of that story's tab order, counted in both engines,
 * so a walk that loses its way anywhere in it is short.
 */
const PROOF: Proof = { story: 'workspace--crowded', stops: 7 }

const preview: Preview = {
  parameters: {
    layout: 'fullscreen',
    controls: { matchers: { color: /(background|colou?r)$/i } },
    backgrounds: { disable: true },
  },
  ...toolbar,
  afterEach: reachCheck(PROOF),
}

export default preview
