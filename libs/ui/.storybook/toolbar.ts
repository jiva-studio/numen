/**
 * The theme, interface size and text size a story is looked at through, as
 * toolbar switches. Shared by the stories of `@numen/ui` and of the plugins.
 */
import type { Preview } from '@storybook/vue3-vite'

/**
 * The two multipliers, as far as each goes. The window refuses a number
 * outside these, so nothing reachable from the toolbar is a size a person
 * could not ask for.
 */
const INTERFACE = [0.8, 1, 1.25, 1.5, 2]
const READING = [0.8, 1, 1.25, 1.5, 1.75]

/** A multiplier as a toolbar item. */
const sizes = (among: readonly number[]) =>
  among.map((size) => ({ value: String(size), title: `${size}×` }))

/** The multiplier a toolbar is standing on. Anything else is as designed. */
const selected = (target: unknown, among: readonly number[]) =>
  among.includes(Number(target)) ? String(Number(target)) : '1'

/**
 * The built stylesheet of `@numen/ui` carries `light-dark()` lowered by
 * Lightning CSS: each pair reads `--lightningcss-light` and
 * `--lightningcss-dark`, which the build sets from `prefers-color-scheme`
 * alone. Stories drawn against the build, as the plugins' are, take the theme
 * from these two. Stories drawn against the source do not read them.
 */
const LOWERED_SCHEME = {
  light: ':root { --lightningcss-light: initial; --lightningcss-dark: ; }',
  dark: ':root { --lightningcss-light: ; --lightningcss-dark: initial; }',
} as const

/** Sets the lowered `light-dark()` of the built stylesheet to one theme. */
const setLoweredScheme = (theme: keyof typeof LOWERED_SCHEME) => {
  let sheet = document.getElementById('story-lowered-scheme')
  if (!sheet) {
    sheet = document.createElement('style')
    sheet.id = 'story-lowered-scheme'
    document.head.append(sheet)
  }
  sheet.textContent = LOWERED_SCHEME[theme]
}

export const toolbar = {
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
      description:
        'How large the interface is drawn: chrome, controls, spacing, and the type in them',
      toolbar: {
        title: 'Interface',
        icon: 'grow',
        items: sizes(INTERFACE),
        dynamicTitle: true,
      },
    },
    font: {
      description:
        'How large the text a person reads is set: a note, a book, an answer, the editor',
      toolbar: {
        title: 'Text',
        icon: 'paragraph',
        items: sizes(READING),
        dynamicTitle: true,
      },
    },
  },
  initialGlobals: { theme: 'light', interface: '1', font: '1' },
  decorators: [
    (story, context) => {
      const theme = context.globals['theme'] === 'dark' ? 'dark' : 'light'
      const root = document.documentElement
      root.style.colorScheme = theme
      setLoweredScheme(theme)
      // On the root, where the window puts them, so every story is looked at
      // through the same two numbers the window is drawn through.
      root.style.setProperty(
        '--numen-interface-scale',
        selected(context.globals['interface'], INTERFACE),
      )
      root.style.setProperty('--numen-text-scale', selected(context.globals['font'], READING))
      return { components: { story }, template: '<story />' }
    },
  ],
} satisfies Pick<Preview, 'globalTypes' | 'initialGlobals' | 'decorators'>
