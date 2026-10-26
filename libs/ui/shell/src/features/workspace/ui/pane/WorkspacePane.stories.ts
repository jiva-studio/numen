/**
 * One pane of the workspace holding an editor among its tabs.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent } from 'storybook/test'
import { ref } from 'vue'
import { Editor } from '@numen/ui'
import { pane } from '../../lib/node'
import WorkspacePane from './WorkspacePane.vue'
import { RUSSIAN } from '@/shared/fixtures/prose'

const meta = {
  title: 'Workspace/Pane',
  component: WorkspacePane,
  parameters: {
    layout: 'fullscreen',
    reach: { keeps: 'Escape hands Tab back to the page' },
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
 * Every tab of a pane is drawn and the ones not shown are held out of sight.
 * The pane says which tab is on screen and that tab's editor is measured again,
 * and the offset the person left the text at is the offset it comes back at.
 */
export const ShownAgain: Story = {
  args: { pane: pane('main', [NOTE_TAB, BESIDE], NOTE_TAB) },
  render: () => ({
    components: { Editor, WorkspacePane },
    setup: () => {
      const heldTab = ref(pane('main', [NOTE_TAB, BESIDE], NOTE_TAB))
      const editors = new Map<string, { measure: () => void }>()
      return {
        held: heldTab,
        NOTE_TAB,
        text: TEXT,
        titles: { [NOTE_TAB]: 'Note', [BESIDE]: 'Beside' },
        choose: (tab: string) => {
          heldTab.value = pane('main', [NOTE_TAB, BESIDE], tab)
        },
        drew: (editor: unknown) => {
          if (editor) editors.set(NOTE_TAB, editor as { measure: () => void })
        },
        shown: (tab: string) => editors.get(tab)?.measure(),
      }
    },
    template: `
      <div class="numen h-screen bg-surface">
        <WorkspacePane :pane="held" :titles="titles" @choose="choose" @show="shown">
          <template #tab="{ id }">
            <Editor
              v-if="id === NOTE_TAB"
              :ref="drew"
              :model-value="text"
              class="h-full"
            />
            <p v-else class="p-4 font-sans text-base text-ink">Something else</p>
          </template>
        </WorkspacePane>
      </div>`,
  }),
  play: async ({ canvasElement }) => {
    const scroller = canvasElement.querySelector('.cm-scroller') as HTMLElement
    const tab = (id: string) =>
      canvasElement.querySelector(`[data-workspace-tab="${id}"]`) as HTMLElement

    scroller.scrollTop = 600
    await nextFrame()
    const offset = scroller.scrollTop
    await expect(offset).toBeGreaterThan(0)

    // Away: the panel is held out of sight, and what has no box has no offset.
    await userEvent.click(tab(BESIDE))
    await nextFrame()
    await expect(scroller.scrollTop).toBe(0)

    await userEvent.click(tab(NOTE_TAB))
    await nextFrame()
    await nextFrame()

    await expect(scroller.scrollTop).toBe(offset)
  },
}
