/**
 * What the editor looks like while it is being typed into. What it does to the
 * text is asserted in `live.test.ts` and `table.test.ts`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, within } from 'storybook/test'
import { createApp, ref } from 'vue'
import { undo } from '@codemirror/commands'
import { EditorSelection } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import Editor from './Editor.vue'
import WorkspacePane from '@/workspace/render/WorkspacePane.vue'
import { pane } from '@/workspace/model'
import { MARKED_UP, TABLE } from '@/fixtures/markdown'
import { ARABIC, DEVANAGARI, LINK, LONG, RUSSIAN, UNBREAKABLE } from '@/fixtures/prose'

const meta = {
  title: 'Text/Editor',
  component: Editor,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'Markdown, written and read in the same place. A construct is drawn ' +
          'as it reads until a selection touches it, and then it is shown as ' +
          'it is written. The text in the editor is the text of the file, ' +
          'mark for mark: put the caret in a heading and the hashes are back ' +
          'where they always were.',
      },
    },
  },
  args: { modelValue: MARKED_UP },
} satisfies Meta<typeof Editor>

export default meta
type Story = StoryObj<typeof meta>
type Render = NonNullable<Story['render']>

const framed =
  (text: string, props: Record<string, unknown> = {}): Render =>
  () => ({
    components: { Editor },
    setup: () => ({ text, props }),
    template: `<Editor :model-value="text" v-bind="props" class="h-screen bg-surface" />`,
  })

/** One of everything: headings, marks, lists, a rule, a table, code and maths. */
export const Playground: Story = { render: framed(MARKED_UP) }

/** The marks as they are written, with nothing drawn for them. */
export const AsWritten: Story = { render: framed(MARKED_UP, { live: false }) }

/** Nothing to type into. */
export const ReadOnly: Story = { render: framed(MARKED_UP, { readonly: true }) }

/**
 * A table on its own. Click a cell and type: what is typed is written back
 * over that cell alone. Tab walks the cells and makes a row at the end,
 * Escape leaves the table. The two buttons along its edges make a row and a
 * column.
 */
export const Table: Story = { render: framed(TABLE) }

/** Nothing written yet, and something to say so. */
export const Empty: Story = { render: framed('', { placeholder: 'Write' }) }

/** No text, and nothing said about there being none. */
export const NothingAtAll: Story = { render: framed('', { placeholder: '' }) }

/** One line, which is what a note is for its first minute. */
export const OneLine: Story = { render: framed('# Entropy\n') }

/** Far too many lines. What the scrolling is for. */
export const FarTooMany: Story = {
  render: framed(
    Array.from({ length: 400 }, (_, index) =>
      index % 8 === 0 ? `## Part ${index / 8 + 1}` : `${index}. ${RUSSIAN}`,
    ).join('\n'),
  ),
}

/** One paragraph past any width, and a quotation of the same. */
export const FarTooLong: Story = {
  render: framed(`${LONG} ${LONG} ${LONG}\n\n> ${LONG}\n`),
}

/** A word and an address with nowhere in them to break. */
export const NothingToBreakAt: Story = {
  render: framed(`${UNBREAKABLE}\n\n- ${UNBREAKABLE}\n- <${LINK}>\n\n\`${UNBREAKABLE}\`\n`),
}

/** Scripts that are not Latin, one that runs the other way, and a word with
 *  nowhere to break. */
export const AwkwardText: Story = {
  render: framed(
    `# ${RUSSIAN}\n\n${DEVANAGARI}\n\n${ARABIC}\n\n- ${UNBREAKABLE}\n- <${LINK}>\n`,
  ),
}

const BODY = Array.from({ length: 120 }, (_, index) => `${index + 1}. ${RUSSIAN}`).join('\n')
const BEFORE = `# Entropy\n\n${BODY}\n`
const AFTER = `# Entropy, retitled by whoever else has this open\n\n${BODY}\n`

/** Where the caret was put before the text was read again. */
const LINE = 40
const COLUMN = 3

const viewOf = (canvas: HTMLElement): EditorView => {
  const view = EditorView.findFromDOM(canvas.querySelector('.editor') as HTMLElement)
  if (!view) throw new Error('the editor drew no view')
  return view
}

const settled = () => new Promise((done) => requestAnimationFrame(() => done(null)))

/** The line at the top of what can be seen. */
const topmost = (view: EditorView): number => {
  const box = view.scrollDOM.getBoundingClientRect()
  const at = view.posAtCoords({ x: box.left + 8, y: box.top + 8 }) ?? 0
  return view.state.doc.lineAt(at).number
}

/**
 * The text read again from under the person: the same file, changed by
 * somebody else while it was open. The caret stays on its line, the scroll
 * offset stays where it was, and an undo does not bring the old text back.
 */
export const ReadAgain: Story = {
  render: () => ({
    components: { Editor },
    setup: () => {
      const text = ref(BEFORE)
      return { text, again: () => (text.value = AFTER) }
    },
    template: `
      <div class="numen flex h-screen flex-col bg-surface">
        <button
          class="shrink-0 border-b border-rule px-3 py-2 text-left font-sans text-small text-ink"
          @click="again"
        >
          Read it again
        </button>
        <Editor v-model="text" class="min-h-0 flex-1" />
      </div>`,
  }),
  play: async ({ canvasElement }) => {
    const view = viewOf(canvasElement)

    view.dispatch({ selection: EditorSelection.single(view.state.doc.line(LINE).from + COLUMN) })
    view.scrollDOM.scrollTop = 600
    await settled()
    await expect(view.scrollDOM.scrollTop).toBeGreaterThan(0)
    const top = topmost(view)

    await userEvent.click(within(canvasElement).getByRole('button'))
    await settled()
    await settled()

    await expect(view.state.doc.toString()).toBe(AFTER)

    const line = view.state.doc.lineAt(view.state.selection.main.head)
    await expect(line.number).toBe(LINE)
    await expect(view.state.selection.main.head - line.from).toBe(COLUMN)
    await expect(topmost(view)).toBe(top)

    undo(view)
    await expect(view.state.doc.toString()).toBe(AFTER)
  },
}


/**
 * What a tab costs while it is open.
 *
 * Every tab of a pane is drawn and hidden, so tab count is live editor count.
 * The heap is reported for docs/performance.md; a threshold here would fail on
 * a browser that collects at a different moment.
 */

export const TenOpenTabs: Story = {
  render: () => ({
    components: { Editor },
    setup: () => ({ text: MARKED_UP }),
    template: `<div class="h-screen overflow-hidden bg-surface" data-tabs>
      <div><Editor :model-value="text" class="h-screen" /></div>
    </div>`,
  }),
  play: async ({ canvasElement }) => {
    const heap = () =>
      (performance as { memory?: { usedJSHeapSize: number } }).memory?.usedJSHeapSize ?? 0
    const mb = (bytes: number) => (bytes / 1024 / 1024).toFixed(1)

    await settled()
    const room = canvasElement.querySelector('[data-tabs]') as HTMLElement
    await expect(room.querySelectorAll('.cm-editor').length).toBe(1)
    const one = heap()

    // Nine more of what a pane holds out of sight, in the page that already
    // has one, so the difference is the tabs and not the page.
    const more = Array.from({ length: 9 }, () => {
      const held = document.createElement('div')
      held.style.display = 'none'
      room.append(held)
      const app = createApp(Editor, { modelValue: MARKED_UP })
      app.mount(held)
      return app
    })
    await settled()
    await settled()
    await expect(room.querySelectorAll('.cm-editor').length).toBe(10)
    const ten = heap()

    console.info(`one open tab: ${mb(one)} MB; ten: ${mb(ten)} MB; each further: ${mb((ten - one) / 9)} MB`)
    more.forEach((app) => app.unmount())
  },
}

/** The two tabs of the pane below, and the one of them holding the note. */
const HELD = 'note'
const BESIDE = 'beside'

/**
 * A tab switched away from and come back to.
 *
 * Every tab of a pane is drawn and the ones not shown are held out of sight.
 * The pane says which tab is on screen and that tab's editor is measured again,
 * and the offset the person left the text at is the offset it comes back at.
 */
export const ShownAgain: Story = {
  render: () => ({
    components: { Editor, WorkspacePane },
    setup: () => {
      const held = ref(pane('main', [HELD, BESIDE], HELD))
      const editors = new Map<string, { measure: () => void }>()
      return {
        held,
        HELD,
        text: BEFORE,
        titles: { [HELD]: 'Note', [BESIDE]: 'Beside' },
        choose: (tab: string) => {
          held.value = pane('main', [HELD, BESIDE], tab)
        },
        drew: (editor: unknown) => {
          if (editor) editors.set(HELD, editor as { measure: () => void })
        },
        shown: (tab: string) => editors.get(tab)?.measure(),
      }
    },
    template: `
      <div class="numen h-screen bg-surface">
        <WorkspacePane :pane="held" :titles="titles" @choose="choose" @show="shown">
          <template #tab="{ id }">
            <Editor
              v-if="id === HELD"
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
    const view = viewOf(canvasElement)
    const tab = (id: string) =>
      canvasElement.querySelector(`[data-workspace-tab="${id}"]`) as HTMLElement

    view.dispatch({ selection: EditorSelection.single(view.state.doc.line(LINE).from + COLUMN) })
    view.scrollDOM.scrollTop = 600
    await settled()
    const offset = view.scrollDOM.scrollTop
    await expect(offset).toBeGreaterThan(0)
    const top = topmost(view)

    // Away: the panel is held out of sight, and what has no box has no offset.
    await userEvent.click(tab(BESIDE))
    await settled()
    await expect(view.scrollDOM.scrollTop).toBe(0)

    await userEvent.click(tab(HELD))
    await settled()
    await settled()

    await expect(view.scrollDOM.scrollTop).toBe(offset)
    await expect(topmost(view)).toBe(top)
  },
}

/**
 * The chord that keeps the text.
 *
 * Put the caret in the text and press Ctrl+S. The editor says the person asked
 * for the text to be kept, the text is left as it was, and the page's own
 * answer to the chord does not run.
 */
export const Kept: Story = {
  render: () => ({
    components: { Editor },
    setup: () => {
      const asked = ref(0)
      const answered = ref('')
      return {
        text: MARKED_UP,
        asked,
        answered,
        kept: () => (asked.value += 1),
        watch: (key: KeyboardEvent) => {
          if (key.key === 's' && (key.ctrlKey || key.metaKey)) {
            answered.value = key.defaultPrevented ? 'answered' : 'left for the page'
          }
        },
      }
    },
    template: `
      <div class="numen flex h-screen flex-col bg-surface" @keydown="watch">
        <p class="shrink-0 border-b border-rule px-3 py-2 font-sans text-small text-ink">
          asked <span data-asked>{{ asked }}</span> times, and
          <span data-answered>{{ answered }}</span>
        </p>
        <Editor :model-value="text" class="min-h-0 flex-1" @save="kept" />
      </div>`,
  }),
  play: async ({ canvasElement }) => {
    const view = viewOf(canvasElement)
    const said = (what: string) =>
      (canvasElement.querySelector(`[${what}]`) as HTMLElement).textContent?.trim()

    await userEvent.click(view.contentDOM)
    await settled()
    await expect(view.hasFocus).toBe(true)

    await userEvent.keyboard('{Control>}s{/Control}')
    await settled()

    await expect(said('data-asked')).toBe('1')
    await expect(said('data-answered')).toBe('answered')
    await expect(view.state.doc.toString()).toBe(MARKED_UP)
  },
}
