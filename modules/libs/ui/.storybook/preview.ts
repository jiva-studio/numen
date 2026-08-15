import type { Preview } from '@storybook/vue3-vite'
import '../src/tokens/tokens.css'
import './preview.css'

const preview: Preview = {
  parameters: {
    layout: 'fullscreen',
    controls: { matchers: { color: /(background|colou?r)$/i } },
    backgrounds: { disable: true },
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
  },
  initialGlobals: { theme: 'light' },
  decorators: [
    (story, context) => {
      const theme = context.globals['theme'] === 'dark' ? 'dark' : 'light'
      document.documentElement.style.colorScheme = theme
      return { components: { story }, template: '<story />' }
    },
  ],
}

export default preview
