import type { Preview } from '@storybook/vue3-vite'
import { configure } from 'storybook/test'
// The one keyboard walk, where it is written. A window draws the components,
// so it is judged by what they are judged by, and a copy of the walk here
// would be a second walk to keep in step with the first.
import { reachCheck, type Proof } from '../../../../libs/ui/.storybook/check'
import '@numen/ui/styles.css'
// The window's own sheet, which every tab in it is drawn under.
import '../src/app.css'
import './preview.css'

// A story waits on a real browser drawing a frame, and two engines draw at
// once on one machine.
configure({ asyncUtilTimeout: 5_000 })

/**
 * The story this window's walk is proved against: the whole window with its
 * tabs open, which is the busiest tab order the editor has. The number is the
 * whole of that order, counted in both engines, so a walk stopping at the hour
 * the review day starts at — and leaving every row under it unreached — is
 * short here.
 */
const PROOF: Proof = { story: 'desktop-window--settings', stops: 25 }

const preview: Preview = {
  parameters: {
    layout: 'fullscreen',
    backgrounds: { disable: true },
  },
  afterEach: reachCheck(PROOF),
  decorators: [
    (story, context) => {
      const theme = context.globals['theme'] === 'dark' ? 'dark' : 'light'
      document.documentElement.style.colorScheme = theme
      return { components: { story }, template: '<story />' }
    },
  ],
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
  },
  initialGlobals: { theme: 'light' },
}

export default preview
