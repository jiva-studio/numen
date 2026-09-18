/**
 * The two multipliers, and the three kinds of length they reach.
 *
 * The toolbar sets both, and every specimen here is drawn from a token, so
 * turning a knob shows which kind moved. It is also where the sizing contract
 * is asserted: the play measures each kind with nothing asked for, at 1, and at
 * either end of what a person can ask for.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'

/**
 * What follows the interface: the chrome, the type it is set in, and the
 * lengths that surround that type.
 */
const CHROME = [
  '--numen-font-size',
  '--numen-edge-label-size',
  '--numen-node-padding',
  '--numen-node-gap',
  '--numen-gutter',
  '--numen-inset',
  '--numen-inset-wide',
  '--numen-radius',
  '--numen-edge-label-halo',
  '--numen-radius-panel',
  '--numen-panel-padding',
  '--numen-action-size',
  '--numen-field-min',
  '--numen-radius-field',
  '--numen-field-text-inset',
  '--numen-turn-gap',
] as const

/** What the text multiplier reaches, over whatever the interface is drawing. */
const READING = ['--numen-reading-size', '--numen-prose-size'] as const

/** What follows neither: one line, whatever it separates. */
const NEITHER = ['--numen-stroke', '--numen-ring-width', '--numen-edge-width'] as const

type Token = (typeof CHROME)[number] | (typeof READING)[number] | (typeof NEITHER)[number]

/** The pixel each is drawn at with nothing asked for, and the root with them. */
const AS_DESIGNED: Record<Token, number> = {
  '--numen-font-size': 13,
  '--numen-edge-label-size': 10,
  '--numen-node-padding': 10,
  '--numen-node-gap': 6,
  '--numen-gutter': 20,
  '--numen-inset': 8,
  '--numen-inset-wide': 12,
  '--numen-radius': 6,
  '--numen-radius-panel': 12,
  '--numen-panel-padding': 12,
  '--numen-action-size': 32,
  '--numen-field-min': 44,
  '--numen-radius-field': 22,
  '--numen-field-text-inset': 14,
  '--numen-turn-gap': 16,
  '--numen-reading-size': 13,
  '--numen-prose-size': 14,
  '--numen-stroke': 1,
  '--numen-ring-width': 2,
  '--numen-edge-width': 1.25,
  '--numen-edge-label-halo': 3,
}

const ROOT_AS_DESIGNED = 16

/** The three kinds, in the words the specimen sheet says them in. */
const KINDS = [
  {
    name: 'The interface',
    says: 'A magnifier over the whole window: chrome, controls, spacing, and the type in them. Each is in rem, and the root’s font size carries them all — the size below with them.',
    tokens: CHROME,
  },
  {
    name: 'The text',
    says: 'The size a note, a book, an answer and the editor are set at. The interface carries it like everything else, and this is a second multiplier over that.',
    tokens: READING,
  },
  {
    name: 'Neither',
    says: 'A line is one line whatever it separates.',
    tokens: NEITHER,
  },
]

const meta = {
  title: 'Tokens/Sizing',
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'How large the interface is drawn, and how large the text a person ' +
          'reads is set. Each is a multiplier, 1 being as designed, and each ' +
          'is on the toolbar above. Every length below is a token, drawn as a ' +
          'bar of itself.',
      },
    },
  },
  render: () => ({
    setup: () => ({ kinds: KINDS }),
    template: `
      <div class="numen h-screen overflow-auto bg-surface p-6 font-sans text-base text-ink">
        <p class="mb-1">The chrome is set in this: a palette row, a menu item, a tab, a button.</p>
        <p class="mb-8" style="font-size:var(--numen-reading-size);line-height:1.6">
          The text a person reads is set in this: a note, a book, an answer, the editor.
        </p>

        <section v-for="kind in kinds" :key="kind.name" class="mb-8">
          <h2 class="mb-1 text-small uppercase text-hushed" style="letter-spacing:var(--numen-caps-tracking)">
            {{ kind.name }}
          </h2>
          <p class="mb-3 text-hushed">{{ kind.says }}</p>
          <div v-for="token in kind.tokens" :key="token" class="mb-1 flex items-center gap-3">
            <code class="w-72 shrink-0 text-small text-hushed">{{ token }}</code>
            <span class="h-2 rounded-pill bg-ink" :style="{ inlineSize: 'var(' + token + ')' }" />
          </div>
        </section>
      </div>
    `,
  }),
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

/** One token's length, in the pixels the page draws it at. */
const measureToken = (token: Token): number => {
  const probe = document.createElement('div')
  probe.style.position = 'fixed'
  probe.style.visibility = 'hidden'
  probe.style.inlineSize = `var(${token})`
  document.body.append(probe)
  const width = probe.getBoundingClientRect().width
  probe.remove()
  return width
}

/** The root's own font size, which is what the interface multiplier sets. */
const root = () => parseFloat(getComputedStyle(document.documentElement).fontSize)

/** Both multipliers as the page is wearing them. An empty string is neither. */
const getScales = () => ({
  interfaceScale: document.documentElement.style.getPropertyValue('--numen-interface-scale'),
  textScale: document.documentElement.style.getPropertyValue('--numen-text-scale'),
})

/** Put the two on the page, the way the served page carries them. */
const wear = (interfaceScale: string, textScale: string) => {
  const style = document.documentElement.style
  for (const [name, size] of [
    ['--numen-interface-scale', interfaceScale],
    ['--numen-text-scale', textScale],
  ] as const) {
    if (size) style.setProperty(name, size)
    else style.removeProperty(name)
  }
}

/**
 * Every token of one kind, against the multiple of itself it should be drawn
 * at. A box is laid out in fractions of a pixel, so each is read to within a
 * twentieth of one.
 */
const showEach = async (tokens: readonly Token[], times: number) => {
  for (const token of tokens) {
    await expect(measureToken(token), token).toBeCloseTo(AS_DESIGNED[token] * times, 1)
  }
}

/**
 * What each of the three kinds is drawn at, as the two multipliers move.
 *
 * The play sets them itself and puts back what the toolbar had, so what is
 * looked at is still whatever the toolbar says.
 */
export const Playground: Story = {
  play: async () => {
    const held = getScales()
    try {
      // Nothing asked for, and 1: the pixel every length resolves to today.
      const asDesigned: readonly (readonly [string, string])[] = [
        ['', ''],
        ['1', '1'],
      ]
      for (const [interfaceScale, textScale] of asDesigned) {
        wear(interfaceScale, textScale)
        await expect(root()).toBe(ROOT_AS_DESIGNED)
        await showEach(CHROME, 1)
        await showEach(READING, 1)
        await showEach(NEITHER, 1)
      }

      // The interface magnifies the window, and what is read is in the window.
      for (const times of [0.8, 1.25, 1.5, 2]) {
        wear(String(times), '')
        await expect(root()).toBeCloseTo(ROOT_AS_DESIGNED * times, 1)
        await showEach(CHROME, times)
        await showEach(READING, times)
        await showEach(NEITHER, 1)
      }

      // The text multiplier moves what is read and leaves the chrome.
      for (const times of [0.8, 1.25, 1.5, 1.75]) {
        wear('', String(times))
        await expect(root()).toBe(ROOT_AS_DESIGNED)
        await showEach(READING, times)
        await showEach(CHROME, 1)
        await showEach(NEITHER, 1)
      }

      // Both at once, at either end: they compose, and every line is the line
      // it was.
      const ends: readonly (readonly [number, number])[] = [
        [0.8, 1.75],
        [2, 0.8],
        [1.5, 1.5],
      ]
      for (const [interfaceScale, textScale] of ends) {
        wear(String(interfaceScale), String(textScale))
        await showEach(NEITHER, 1)
        await showEach(CHROME, interfaceScale)
        await showEach(READING, interfaceScale * textScale)
      }
    } finally {
      wear(held.interfaceScale, held.textScale)
    }
  },
}
