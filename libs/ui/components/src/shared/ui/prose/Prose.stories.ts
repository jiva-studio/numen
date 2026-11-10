/**
 * Everything an agent is likely to write, typeset the way the panel shows it.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import Prose from './Prose.vue'

const onFollow = fn()

const meta = {
  title: 'Chat/Prose',
  component: Prose,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'Markdown, turned into elements by the renderer and typeset by the ' +
          'typography plugin. HTML in the text stays text.',
      },
    },
  },
  decorators: [
    (story) => ({
      components: { story },
      template: `
        <div class="numen w-[420px] max-w-[calc(100vw-2rem)] rounded-panel border border-rule bg-surface px-4 py-3 text-answer-ink">
          <story />
        </div>
      `,
    }),
  ],
  args: { text: '' },
} satisfies Meta<typeof Prose>

export default meta
type Story = StoryObj<typeof meta>

export const Playground: Story = {
  args: {
    text: [
      'Three notes are about **entropy**, and one of them is `Entropy` itself.',
      '',
      'What I would do:',
      '',
      '1. read the parent',
      '2. add the missing children',
      '3. leave the frontmatter alone',
    ].join('\n'),
  },
}

/** A heading, a quote, a rule and a table — the rest of the vocabulary. */
export const Everything: Story = {
  args: {
    text: [
      '## What is here',
      '',
      '> A note is named by its file.',
      '',
      '| note | seat |',
      '| --- | --- |',
      '| Entropy | child |',
      '| Thermodynamics | parent |',
      '',
      '---',
      '',
      'And a [link](https://example.com) that goes nowhere.',
    ].join('\n'),
  },
}

/** Code, which arrives whenever the agent quotes what it wrote. */
export const Code: Story = {
  args: {
    text: [
      'Written like this:',
      '',
      '```markdown',
      '# Simple pendulum',
      '',
      'part of: [[Harmonic oscillator]]',
      '```',
    ].join('\n'),
  },
}

/** Half a sentence: prose is marked up again on every piece that arrives. */
export const HalfWritten: Story = {
  args: { text: 'The style is clear: a `# Title` line, one evocative sen' },
}

/**
 * Text written by another user is rendered verbatim, not interpreted as
 * markup.
 */
export const NotMarkup: Story = {
  args: { text: 'A note can say <script>alert(1)</script> and it is prose, not a program.' },
}

/**
 * A pressed link is reported with its address. A link the caller named is held back from the
 * browser; any other link is left to it.
 */
export const ReportsAPressedLink: Story = {
  args: {
    text: 'Tools are kept in [[Tool storage]], and there is [a guide](#guide) to the shed.',
    links: [{ raw: '[[Tool storage]]', href: 'garden://tool-storage', text: 'Tool storage' }],
    onFollow,
  },
  beforeEach: () => {
    onFollow.mockClear()
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await userEvent.click(canvas.getByRole('link', { name: 'Tool storage' }))
    await expect(onFollow).toHaveBeenCalledTimes(1)
    const [named, namedPress] = onFollow.mock.calls[0] as [string, MouseEvent]
    await expect(named).toBe('garden://tool-storage')
    await expect(namedPress.defaultPrevented).toBe(true)

    await userEvent.click(canvas.getByRole('link', { name: 'a guide' }))
    await expect(onFollow).toHaveBeenCalledTimes(2)
    const [other, otherPress] = onFollow.mock.calls[1] as [string, MouseEvent]
    await expect(other).toBe('#guide')
    await expect(otherPress.defaultPrevented).toBe(false)
  },
}

/** A press on text that is not a link reports nothing. */
export const IgnoresAPressOnText: Story = {
  args: { text: 'Tools are kept in the shed.\n\nSee [a guide](#guide).', onFollow },
  beforeEach: () => {
    onFollow.mockClear()
  },
  play: async ({ canvasElement }) => {
    await userEvent.click(within(canvasElement).getByText('Tools are kept in the shed.'))
    await expect(onFollow).not.toHaveBeenCalled()
  },
}
