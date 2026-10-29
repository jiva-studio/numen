/**
 * An icon drawn by the Lucide name a manifest gives it, in the size and colour of the line it
 * is drawn in.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import NamedIcon from './NamedIcon.vue'

const meta = {
  title: 'NamedIcon',
  component: NamedIcon,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'An icon a plugin names in its manifest, such as `trash-2`. The set of ' +
          'icons loads as one file the first time an icon is drawn, so a name ' +
          'is drawn a moment after it is given, and a name that is none draws ' +
          'nothing.',
      },
    },
  },
} satisfies Meta<typeof NamedIcon>

export default meta
type Story = StoryObj<typeof meta>

/** An icon the manifests use, beside the text it belongs to. */
export const Playground: Story = {
  args: { name: 'trash-2' },
  render: () => ({
    components: { NamedIcon },
    template: `
      <div class="numen" style="padding:24px;background:var(--numen-surface);color:var(--numen-ink);font:var(--numen-font-size)/var(--numen-line-height) var(--numen-font-sans)">
        <span style="display:inline-flex;align-items:center;gap:8px">
          <NamedIcon name="trash-2" style="inline-size:1rem;block-size:1rem" />
          <span>Delete transcript</span>
        </span>
      </div>
    `,
  }),
}

/** A name that is none draws nothing, and leaves the text where it was. */
export const UnknownName: Story = {
  args: { name: 'no-such-icon' },
  render: () => ({
    components: { NamedIcon },
    template: `
      <div class="numen" style="padding:24px;background:var(--numen-surface);color:var(--numen-ink);font:var(--numen-font-size)/var(--numen-line-height) var(--numen-font-sans)">
        <span style="display:inline-flex;align-items:center;gap:8px">
          <NamedIcon name="no-such-icon" style="inline-size:1rem;block-size:1rem" />
          <span>Nothing before this</span>
        </span>
      </div>
    `,
  }),
}
