/**
 * A mark saying more is on its way, on the grounds it is drawn on and beside
 * the text it takes its size from.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import Waiting from './Waiting.vue'

const meta = {
  title: 'Generic/Waiting',
  component: Waiting,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Something is still filling and more of it is on its way. It says ' +
          'nothing about how much is left: what is coming is counted ' +
          'somewhere or it is not, and a length that guesses is worse than ' +
          'one making no claim. It takes the size of the line it stands on ' +
          'and the colour of whatever it stands in.',
      },
    },
  },
} satisfies Meta<typeof Waiting>

export default meta
type Story = StoryObj<typeof meta>

/** On its own, at the size of ordinary text. */
export const Playground: Story = {
  render: () => ({
    components: { Waiting },
    template: `
      <div class="numen" style="padding:24px;background:var(--numen-surface);color:var(--numen-ink);font:var(--numen-font-size)/var(--numen-line-height) var(--numen-font-sans)">
        <Waiting />
      </div>
    `,
  }),
}

/**
 * Beside the small print it belongs to, taking that print's size and colour
 * rather than the size and colour of the text above it.
 */
export const BesideAName: Story = {
  render: () => ({
    components: { Waiting },
    template: `
      <div class="numen" style="inline-size:280px;padding:24px;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans)">
        <p style="display:flex;align-items:center;gap:6px;margin:0 0 6px;color:var(--numen-edge-label);font-size:var(--numen-edge-label-size);letter-spacing:var(--numen-caps-tracking);text-transform:uppercase">
          <span>Meaning</span>
          <Waiting />
        </p>
        <p style="margin:0;font-size:var(--numen-font-size)">The arrow of time</p>
      </div>
    `,
  }),
}

/** On a filled ground, where it has to read against the colour under it. */
export const OnAFilledGround: Story = {
  render: () => ({
    components: { Waiting },
    template: `
      <div class="numen" style="padding:24px;background:var(--numen-accent);color:var(--numen-accent-ink);font:var(--numen-font-size)/var(--numen-line-height) var(--numen-font-sans)">
        <span style="display:inline-flex;align-items:center;gap:8px">
          <span>Reading the vault</span>
          <Waiting />
        </span>
      </div>
    `,
  }),
}
