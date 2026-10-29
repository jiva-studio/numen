/**
 * One pane of the workspace holding a scrolled text among its tabs.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent } from 'storybook/test'
import { ref } from 'vue'
import { pane } from '../../lib/node'
import WorkspacePane from './WorkspacePane.vue'
import { RUSSIAN } from '@/shared/fixtures/prose'

const meta = {
  title: 'Workspace/Pane',
  component: WorkspacePane,
  parameters: {
    layout: 'fullscreen',
  },
} satisfies Meta<typeof WorkspacePane>

export default meta
type Story = StoryObj<typeof meta>

/** The two tabs of the pane, and the one of them holding the note. */
const NOTE_TAB = 'note'
const BESIDE = 'beside'

const TEXT = `# Entropy\n\n${Array.from({ length: 120 }, (_, index) => `${index + 1}. ${RUSSIAN}`).join('\n')}\n`

const nextFrame = () => new Promise((isDone) => requestAnimationFrame(() => isDone(null)))

/**
 * A tab switched away from and come back to.
 *
 * Every tab of a pane is rendered and the inactive ones are hidden. The pane
 * signals which tab is visible again, so the tab can remeasure, and the text in
 * it keeps what was typed.
 */
export const ShownAgain: Story = {
  args: { pane: pane('main', [NOTE_TAB, BESIDE], NOTE_TAB) },
  render: () => ({
    components: { WorkspacePane },
    setup: () => {
      const pendingTab = ref(pane('main', [NOTE_TAB, BESIDE], NOTE_TAB))
      const shownTabs = ref<string[]>([])
      return {
        held: pendingTab,
        NOTE_TAB,
        text: TEXT,
        shownTabs,
        titles: { [NOTE_TAB]: 'Note', [BESIDE]: 'Beside' },
        choose: (tab: string) => {
          pendingTab.value = pane('main', [NOTE_TAB, BESIDE], tab)
        },
        onShow: (tab: string) => shownTabs.value.push(tab),
      }
    },
    template: `
      <div class="numen h-screen bg-surface">
        <WorkspacePane :pane="held" :titles="titles" @choose="choose" @show="onShow">
          <template #tab="{ id }">
            <textarea
              v-if="id === NOTE_TAB"
              :value="text"
              aria-label="Note"
              class="h-full w-full resize-none p-4 font-sans text-base text-ink"
            />
            <p v-else class="p-4 font-sans text-base text-ink">Something else</p>
          </template>
        </WorkspacePane>
        <output data-shown>{{ shownTabs.join(' ') }}</output>
      </div>`,
  }),
  play: async ({ canvasElement }) => {
    const note = canvasElement.querySelector('textarea') as HTMLTextAreaElement
    const shownLog = canvasElement.querySelector('[data-shown]') as HTMLElement
    const tab = (id: string) =>
      canvasElement.querySelector(`[data-workspace-tab="${id}"]`) as HTMLElement

    await userEvent.type(note, 'Kept.')

    await userEvent.click(tab(BESIDE))
    await nextFrame()
    await userEvent.click(tab(NOTE_TAB))
    await nextFrame()

    await expect(shownLog.textContent?.split(' ').at(-1)).toBe(NOTE_TAB)
    await expect(note.value).toContain('Kept.')
  },
}
