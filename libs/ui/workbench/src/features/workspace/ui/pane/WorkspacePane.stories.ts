/**
 * One pane of the workspace holding a scrolled text among its tabs.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent } from 'storybook/test'
import { computed, provide, ref } from 'vue'
import { pane, type Tab, type TabId } from '../../lib/node'
import { NO_STRIP_EDGE, WORKSPACE_CONTEXT, type WorkspaceContext } from '../../model/context'
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
 * A tab switched away from and returned to.
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
        current: pendingTab,
        NOTE_TAB,
        text: ref(TEXT),
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
        <WorkspacePane :pane="current" :titles="titles" @choose="choose" @show="onShow">
          <template #tab="{ id }">
            <textarea
              v-if="id === NOTE_TAB"
              v-model="text"
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
    await expect(note.isConnected).toBe(true)
    await expect(note.value).toContain('Kept.')
  },
}

/** A tab shows no tooltip for its title; hovering its mark shows the mark tooltip. */
export const TabNamedByTooltip: Story = {
  args: { pane: pane('main', [NOTE_TAB, BESIDE], NOTE_TAB) },
  render: (args) => ({
    components: { WorkspacePane },
    setup: () => {
      const titles: Readonly<Record<string, string>> = { [NOTE_TAB]: 'Note', [BESIDE]: 'Beside' }
      provide(
        WORKSPACE_CONTEXT,
        computed<WorkspaceContext>(() => ({
          findTab: (id: TabId): Tab | undefined =>
            titles[id] === undefined
              ? undefined
              : { id, title: titles[id], ...(id === BESIDE ? { mark: 'unsaved' } : {}) },
          focus: 'main',
          minimum: 220,
          getStripEdge: () => NO_STRIP_EDGE,
          choose: fn(),
          close: fn(),
          startTabPress: fn(),
          openMenu: fn(),
          claim: fn(),
          resize: fn(),
          show: fn(),
        })),
      )
      return { args }
    },
    template: `
      <div class="numen h-screen bg-surface">
        <WorkspacePane :pane="args.pane" />
      </div>`,
  }),
  play: async ({ canvasElement }) => {
    const getTab = (id: TabId) =>
      canvasElement.querySelector(`[data-workspace-tab="${id}"]`) as HTMLElement
    // The tooltip an element points at, so one that another story left in the body does not count.
    const getTooltip = (element: HTMLElement) => {
      const id = element.getAttribute('aria-describedby')
      return id ? document.getElementById(id) : null
    }
    const tab = getTab(BESIDE)
    const mark = tab.querySelector('.tab__mark') as HTMLElement

    await userEvent.unhover(tab)
    await expect(tab.querySelector('[title]')).toBeNull()
    await expect(getTooltip(tab)).toBeNull()

    await userEvent.hover(tab)
    await expect(getTooltip(tab)).toBeNull()

    await userEvent.hover(mark)
    await expect(getTooltip(mark)).toHaveTextContent('unsaved')

    await userEvent.unhover(mark)
    await expect(getTooltip(mark)).toBeNull()

    await userEvent.keyboard('{Tab}')
    await expect(getTooltip(getTab(NOTE_TAB))).toBeNull()
  },
}
