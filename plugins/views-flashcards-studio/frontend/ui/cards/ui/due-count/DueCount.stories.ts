/**
 * How many cards are due, as one figure in a pill. Also the test corpus: each
 * story is run in a browser by `@storybook/addon-vitest`.
 *
 * The pill has two grounds. On its own it takes a token; on a filled button it
 * is drawn from the button's own text, and only a browser resolves that.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, within } from 'storybook/test'
import DueCount from './DueCount.vue'
import { Button } from '@numen/ui'
import { DUE_WORDS } from './due'
import { lightness } from '../../../fixtures/colour'
import { DARK, expectDark } from '../../../fixtures/theme'

interface Knobs {
  /** Cards due today. Nothing until it has been counted. */
  due: number | null
  isBare: boolean
  isOnButton: boolean
}

/** The pill on the surface, and the same pill on a filled button. */
const renderPills = (args: Knobs) => ({
  components: { DueCount, Button },
  setup: () => ({ args }),
  template: `
    <div class="numen" style="display:flex;align-items:center;gap:24px;padding:32px;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
      <span style="display:inline-flex;align-items:center;gap:8px">
        Sanskrit
        <DueCount :due="args.due" :is-bare="args.isBare" />
      </span>
      <Button variant="solid">
        Review
        <DueCount :due="args.due" :is-bare="args.isBare" is-on-button />
      </Button>
    </div>
  `,
})

const meta: Meta<Knobs> = {
  title: 'Flash Cards/Due count',
  component: DueCount,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'One figure and the word for what it counts. A figure not worked ' +
          'out yet is drawn as the shape it will be, in the same box, so the ' +
          'row it stands in does not move when it lands.',
      },
    },
  },
  argTypes: {
    due: { control: 'number' },
    isBare: { control: 'boolean' },
    isOnButton: { control: 'boolean' },
  },
  args: { due: 12, isBare: false, isOnButton: false },
  render: renderPills,
}

export default meta
type Story = StoryObj<Knobs>

/** A figure that has landed, on the surface and on a filled button. */
export const Playground: Story = {}

/** The figure alone, which is what a long list of rows draws. */
export const Bare: Story = {
  args: { isBare: true },
  play: async ({ canvasElement }) => {
    const [pill] = within(canvasElement).getAllByRole('status')
    expect(pill?.textContent?.trim()).toBe('12')

    // Drawn as the figure alone, and still read out as what it counts.
    expect(pill?.getAttribute('aria-label')).toBe(DUE_WORDS.formatDue(12))
  },
}

/**
 * Still being counted: the pill holds the shape the figure will take, and says
 * so to a reader who is listening.
 */
export const StillCounting: Story = {
  args: { due: null },
  play: async ({ canvasElement }) => {
    const [pill] = within(canvasElement).getAllByRole('status')
    expect(pill?.getAttribute('aria-label')).toBe(DUE_WORDS.counting)
    expect(pill?.textContent?.trim()).toBe('')
  },
}

/**
 * On the dark set of tokens, on the surface and on a filled button.
 *
 * The pill on a button is drawn from the button's own text, so its ground has
 * to come to rest between the button's ground and that text — on the dark set
 * as on the light one, where the two are the other way about.
 */
export const Dark: Story = {
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    const canvas = within(canvasElement)
    const [plain, overflow] = canvas.getAllByRole('status')
    const button = canvas.getByRole('button')
    expect(plain).toBeDefined()
    expect(overflow).toBeDefined()

    // On its own the pill carries both of its own colours, and they are told
    // apart: its ground is thin, so it is read over the surface behind it, and
    // on the dark set the figure stands above that ground.
    const alone = getComputedStyle(plain!)
    const backdropColor = getComputedStyle(
      canvasElement.querySelector('.numen') as HTMLElement,
    ).backgroundColor
    expect(lightness(alone.color)).toBeGreaterThan(
      lightness(alone.backgroundColor, backdropColor) + 24,
    )

    // On the button it carries neither: the ink is the button's, and the
    // ground is that ink laid thinly over the button's own.
    const filled = getComputedStyle(button)
    const ink = lightness(filled.color)
    const ground = lightness(filled.backgroundColor)
    expect(Math.abs(ink - ground)).toBeGreaterThan(24)

    const pill = lightness(getComputedStyle(overflow!).backgroundColor, filled.backgroundColor)
    expect(pill).toBeGreaterThan(Math.min(ink, ground))
    expect(pill).toBeLessThan(Math.max(ink, ground))
    expect(Math.abs(pill - ground)).toBeGreaterThan(2)
  },
}

/**
 * The theme the window is set to, drawn against the opposite one the machine prefers. The tokens
 * of the built `@numen/ui` follow `color-scheme` on the root, which is where the window writes
 * its mode, and not `prefers-color-scheme`.
 */
const drawAgainstSystem =
  (system: 'light' | 'dark', isDark: boolean): NonNullable<Story['play']> =>
  async ({ canvasElement }) => {
    const { commands } = await import('vitest/browser')
    await commands.setSystemScheme(system)
    try {
      const ground = canvasElement.querySelector('.numen')
      if (!ground) throw new Error('no ground drawn')
      const shade = lightness(getComputedStyle(ground).backgroundColor)
      if (isDark) expect(shade).toBeLessThan(64)
      else expect(shade).toBeGreaterThan(192)
    } finally {
      await commands.setSystemScheme(null)
    }
  }

/** The dark theme on a machine that prefers light. */
export const DarkOnLightSystem: Story = {
  globals: DARK,
  play: drawAgainstSystem('light', true),
}

/** The light theme on a machine that prefers dark. */
export const LightOnDarkSystem: Story = {
  globals: { theme: 'light' },
  play: drawAgainstSystem('dark', false),
}
