/**
 * The contrast of the name of a tab showing and of one that is not, in the light set and in the dark
 * one. Also the test corpus: a browser resolves each colour as drawn, and the story fails when the
 * name of the showing tab falls below 4.5:1 on the tab.
 *
 * Each reading is printed as a `CONTRAST|kind|name|ratio` line.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect } from 'storybook/test'
import WorkspaceTab from './WorkspaceTab.vue'
import { DARK, expectDark } from '@/shared/fixtures/theme'
import { expectAllText } from '../../../../../.storybook/contrast'

const meta: Meta = {
  title: 'Application/Workspace tab contrast',
  parameters: { layout: 'fullscreen' },
}

export default meta
type Story = StoryObj

const strip = () => ({
  components: { WorkspaceTab },
  template: `
    <div class="numen" style="display:flex;gap:4px;padding:8px;background:var(--numen-surface);color:var(--numen-ink);font-family:var(--numen-font-sans)">
      <WorkspaceTab tab="a" title="Entropy" is-showing is-focused />
      <WorkspaceTab tab="b" title="Heat engines" />
      <WorkspaceTab tab="c" title="Carnot cycle" is-showing />
    </div>`,
})

async function playTabs(canvasElement: HTMLElement): Promise<void> {
  const tabs = canvasElement.querySelectorAll('[role="tab"]')
  await expect(tabs.length).toBe(3)
  for (const tab of tabs)
    await expectAllText(
      tab,
      `tab ${tab.getAttribute('aria-selected') === 'true' ? 'showing' : 'not showing'}`,
    )
}

export const Light: Story = {
  render: strip,
  play: async ({ canvasElement }) => playTabs(canvasElement),
}

export const Dark: Story = {
  render: strip,
  globals: DARK,
  play: async ({ canvasElement }) => {
    await expectDark(canvasElement)
    await playTabs(canvasElement)
  },
}
