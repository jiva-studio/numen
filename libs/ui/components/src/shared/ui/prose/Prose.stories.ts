/**
 * Everything a writer is likely to produce, typeset the way the panel shows it.
 */
import type { Decorator, Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, within } from 'storybook/test'
import Prose from './Prose.vue'

/** Whether the browser was still free to follow each pressed link when it was reported. */
const wasPrevented: boolean[] = []
const onFollow = fn((_href: string, press: MouseEvent) => {
  wasPrevented.push(press.defaultPrevented)
})

const keepLinksInPlace: Decorator = (story) => ({
  components: { story },
  template: '<div @click.prevent><story /></div>',
})

const meta = {
  title: 'Text/Prose',
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
      'Three pages are about **entropy**, and one of them is `Entropy` itself.',
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
      '> A page is named by its file.',
      '',
      '| page | seat |',
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

/** Code, which arrives whenever the writer quotes what it wrote. */
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
  args: { text: 'A page can say <script>alert(1)</script> and it is prose, not a program.' },
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
  decorators: [keepLinksInPlace],
  beforeEach: () => {
    onFollow.mockClear()
    wasPrevented.length = 0
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await userEvent.click(canvas.getByRole('link', { name: 'Tool storage' }))
    await expect(onFollow).toHaveBeenCalledTimes(1)
    await expect(onFollow.mock.calls[0]?.[0]).toBe('garden://tool-storage')
    await expect(wasPrevented[0]).toBe(true)

    await userEvent.click(canvas.getByRole('link', { name: 'a guide' }))
    await expect(onFollow).toHaveBeenCalledTimes(2)
    await expect(onFollow.mock.calls[1]?.[0]).toBe('#guide')
    await expect(wasPrevented[1]).toBe(false)
  },
}

/** A press on text that is not a link reports nothing. */
export const IgnoresAPressOnText: Story = {
  args: { text: 'Tools are kept in the shed.\n\nSee [a guide](#guide).', onFollow },
  decorators: [keepLinksInPlace],
  beforeEach: () => {
    onFollow.mockClear()
    wasPrevented.length = 0
  },
  play: async ({ canvasElement }) => {
    await userEvent.click(within(canvasElement).getByText('Tools'))
    await expect(onFollow).not.toHaveBeenCalled()
  },
}
