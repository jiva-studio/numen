/**
 * The shape a value will take, held open while the value is worked out.
 *
 * The stories are a list filling row by row, because that is where the claim
 * lives: a shape the size of what replaces it leaves the rows where they are.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import Coming from './Coming.vue'

const meta = {
  title: 'Generic/Coming',
  component: Coming,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'A filled shape standing in the room a value will take while that ' +
          'value is being worked out. It is drawn at the size of what ' +
          'replaces it, so nothing moves when the value lands, and it says ' +
          'that something is coming.',
      },
    },
  },
  argTypes: {
    wide: { control: 'text' },
    high: { control: 'text' },
    pill: { control: 'boolean' },
  },
  args: { wide: '1.5rem', high: '0.75em', pill: true },
} satisfies Meta<typeof Coming>

export default meta
type Story = StoryObj<typeof meta>

/** On its own, at the size a count of two or three figures takes. */
export const Playground: Story = {}

/**
 * The claim, drawn twice: a list where every count has arrived, and the same
 * list where none has. The rows stand in the same places in both.
 */
export const ARowThatDoesNotMove: Story = {
  render: (args) => ({
    components: { Coming },
    setup: () => ({ args, rows: ['Studies', 'Sanskrit', 'Птицы', 'ᬩᬮᬶ'] }),
    template: `
      <div class="numen" style="display:flex;gap:32px;padding:24px;background:var(--numen-surface);color:var(--numen-node-fg);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <ul v-for="(counted, at) in [true, false]" :key="at" style="margin:0;padding:0;list-style:none;inline-size:11rem">
          <li v-for="(name, row) in rows" :key="name" style="display:flex;align-items:center;gap:12px;padding:0.3rem 0.6rem">
            <span style="flex:1">{{ name }}</span>
            <span v-if="counted" style="flex:none;min-inline-size:1.5rem;padding:0.0625rem 0.4rem;text-align:center;border-radius:var(--numen-radius-pill);background:var(--numen-highlight);color:var(--numen-caution-fg);font-size:var(--numen-edge-label-size);font-variant-numeric:tabular-nums">{{ [7, 128, 0, 42][row] }}</span>
            <span v-else style="flex:none;min-inline-size:1.5rem;padding:0.0625rem 0.4rem;text-align:center;border-radius:var(--numen-radius-pill);background:var(--numen-highlight);color:var(--numen-caution-fg);font-size:var(--numen-edge-label-size)"><Coming v-bind="args" /></span>
          </li>
        </ul>
      </div>
    `,
  }),
}

/** A line of text, where what is coming is a name. */
export const InPlaceOfWords: Story = {
  args: { wide: '9rem', high: '1em', pill: false },
  render: (args) => ({
    components: { Coming },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="inline-size:280px;padding:24px;background:var(--numen-surface);color:var(--numen-node-fg);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <p style="margin:0 0 6px;color:var(--numen-edge-label);font-size:var(--numen-edge-label-size)">Preset</p>
        <Coming v-bind="args" />
      </div>
    `,
  }),
}

/** Filling the whole of what holds it, which is what it does given no width. */
export const AsWideAsWhatHoldsIt: Story = {
  args: { wide: '100%', high: '3rem', pill: false },
  render: (args) => ({
    components: { Coming },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="inline-size:320px;padding:24px;background:var(--numen-surface);color:var(--numen-node-fg);font-family:var(--numen-font-sans)">
        <Coming v-bind="args" />
      </div>
    `,
  }),
}

/** On a filled ground, where the fill has to read against the colour under it. */
export const OnAFilledGround: Story = {
  args: { wide: '4rem', high: '1em', pill: true },
  render: (args) => ({
    components: { Coming },
    setup: () => ({ args }),
    template: `
      <div class="numen" style="padding:24px;background:var(--numen-focus-bg);color:var(--numen-focus-fg);font-family:var(--numen-font-sans);font-size:var(--numen-font-size)">
        <span style="display:inline-flex;align-items:center;gap:8px">
          <span>Counting</span>
          <Coming v-bind="args" />
        </span>
      </div>
    `,
  }),
}

/** Far too many of them at once, which is a list nothing has answered for yet. */
export const FarTooMany: Story = {
  args: { wide: '100%', high: '0.9rem', pill: false },
  render: (args) => ({
    components: { Coming },
    setup: () => ({ args, rows: Array.from({ length: 40 }, (_, at) => at) }),
    template: `
      <div class="numen" style="display:flex;flex-direction:column;gap:6px;inline-size:320px;padding:24px;background:var(--numen-surface);color:var(--numen-node-fg)">
        <Coming v-for="row in rows" :key="row" v-bind="args" />
      </div>
    `,
  }),
}
