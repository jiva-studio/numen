/**
 * What the editor looks like while it is being typed into. What it does to the
 * text is asserted in `live.test.ts`, `table.test.ts` and `change.test.ts`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { createApp, ref } from 'vue'
import { undo } from '@codemirror/commands'
import { EditorSelection } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import Editor from './Editor.vue'
import type { EditorChange } from './change'
import WorkspacePane from '@/workspace/render/WorkspacePane.vue'
import { pane } from '@/workspace/node'
import { MARKED_UP, PICTURE, TABLE } from '@/fixtures/markdown'
import { ARABIC, DEVANAGARI, LINK, LONG, RUSSIAN, UNBREAKABLE } from '@/fixtures/prose'

const meta = {
  title: 'Text/Editor',
  component: Editor,
  parameters: {
    layout: 'fullscreen',
    // Tab indents while the editor holds it, and Escape hands it back to the
    // page. What a reader is told on arrival is that way out.
    reach: { keeps: 'Escape hands Tab back to the page' },
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

/* One line of every construct whose marks are concealed, each with a blank
   line after it, and a line at the end to park the caret on. */
const BULLET = '- a bullet'
const TASK = '- [ ] something to do'
const CONCEALED = [
  '# Heading one',
  '## Heading two',
  '### Heading three',
  '#### Heading four',
  '##### Heading five',
  '###### Heading six',
  'A **bold** and *slanted* and ~~struck~~ word.',
  'A `snippet` and a [somewhere](https://example.invalid/x "a title").',
  '> A quotation.',
  BULLET,
  TASK,
]

/* Two constructs drawn as something other than text. What is drawn takes its
   own height, which is not the height of the line it was written on. */
const REDRAWN = ['---', `![a picture](${PICTURE})`]

const EVERY = [...CONCEALED, ...REDRAWN]
const STEADY = `${EVERY.join('\n\n')}\n\nSomething well away from all of it.\n`

/** Which line a construct is written on, and the line nothing is drawn on. */
const written = (text: string) => EVERY.indexOf(text) * 2 + 1
const PARKED = EVERY.length * 2 + 1

/* A heading is set at the tightest line in the editor, and how far a letter
   stands above and below its baseline is the font's answer. These are the
   machine's own sans cut twice: one face reaching past that line, one well
   inside it. */
const SANS =
  "local('DejaVu Sans'), local('Liberation Sans'), local('Noto Sans'), " +
  "local('FreeSans'), local('Nimbus Sans'), local('Arial'), local('Helvetica'), " +
  "local('Cantarell'), local('Ubuntu'), local('Roboto'), local('Verdana')"
const TALL = { name: 'numen-tall', size: '150%' }
const LOW = { name: 'numen-low', size: '60%' }
const FACES = [TALL, LOW]

/** The two faces, handed to the page. */
const cutting = () => {
  const sheet = document.createElement('style')
  sheet.textContent = FACES.map(
    ({ name, size }) => `@font-face { font-family: '${name}'; src: ${SANS}; size-adjust: ${size} }`,
  ).join('\n')
  document.head.append(sheet)
  return Promise.all(FACES.map(({ name }) => document.fonts.load(`100px '${name}'`)))
}

/** How tall a line of a face is when nothing but the face decides. */
const reach = (name: string) => {
  const probe = document.createElement('span')
  probe.textContent = 'x'
  probe.style.cssText =
    'position:absolute;visibility:hidden;line-height:normal;font-size:100px;' +
    `font-family:'${name}'`
  document.body.append(probe)
  const height = probe.getBoundingClientRect().height
  probe.remove()
  return height
}

/**
 * The line a construct is written on, measured with the marks concealed and
 * with them back. Text drawn as text keeps its height; a rule and a picture
 * are drawn as themselves and take their own. Every measurement is made under
 * the machine's font and under both cut faces.
 */
export const Steady: Story = {
  render: framed(STEADY),
  play: async ({ canvasElement }) => {
    const view = viewOf(canvasElement)

    const put = async (line: number, column: number) => {
      view.dispatch({ selection: EditorSelection.single(view.state.doc.line(line).from + column) })
      await settled()
      await settled()
    }

    /** The block the editor drew for a line, measured where it stands. */
    const heightOf = (line: number) => {
      const row = view.state.doc.line(line)
      const drawn = Array.from(view.contentDOM.children).find((block) => {
        const at = view.posAtDOM(block)
        return at >= row.from && at <= row.to
      })
      if (!drawn) throw new Error(`nothing is drawn for line ${line}`)
      return drawn.getBoundingClientRect().height
    }

    await cutting()
    // A face the machine cannot cut pins nothing.
    await expect(reach(TALL.name) > reach(LOW.name)).toBe(true)

    for (const chosen of [null, ...FACES]) {
      const face = chosen ? chosen.name : 'the machine'
      if (chosen) view.dom.style.setProperty('--numen-font-sans', `'${chosen.name}'`)
      else view.dom.style.removeProperty('--numen-font-sans')
      await settled()

      for (const text of CONCEALED) {
        const line = written(text)
        await put(PARKED, 0)
        const concealed = heightOf(line)
        await put(line, 2)
        await expect({ face, text, height: heightOf(line) }).toEqual({
          face,
          text,
          height: concealed,
        })
      }

      for (const text of REDRAWN) {
        const line = written(text)
        await put(PARKED, 0)
        const concealed = heightOf(line)
        await put(line, 2)
        await expect({ face, text, same: heightOf(line) === concealed }).toEqual({
          face,
          text,
          same: false,
        })
      }

      // Either side of what stands in place of a mark there is somewhere for
      // the caret to be drawn, and a click there lands on that line.
      await put(PARKED, 0)
      for (const [text, column] of [
        ['# Heading one', 0],
        ['# Heading one', 2],
        [BULLET, 0],
        [BULLET, 1],
        [TASK, 0],
        [TASK, 1],
      ] as const) {
        const line = written(text)
        const spot = view.coordsAtPos(view.state.doc.line(line).from + column)
        const back = spot
          ? view.posAtCoords({ x: spot.left + 1, y: (spot.top + spot.bottom) / 2 })
          : null
        await expect({
          face,
          text,
          column,
          drawn: !!spot && spot.bottom - spot.top > 0,
          lands: back !== null && view.state.doc.lineAt(back).number === line,
        }).toEqual({ face, text, column, drawn: true, lands: true })
      }

      // The task's box stands inside the line it marks.
      const box = canvasElement.querySelector('.cm-box') as HTMLElement
      const held = box.getBoundingClientRect()
      const around = (box.closest('.cm-line') as HTMLElement).getBoundingClientRect()
      await expect({
        face,
        inside: held.top >= around.top && held.bottom <= around.bottom,
      }).toEqual({ face, inside: true })
    }
  },
}

/**
 * A table on its own. Click a cell and type: what is typed is written back
 * over that cell alone. Tab walks the cells and makes a row at the end,
 * Escape leaves the table. The two buttons along its edges make a row and a
 * column.
 */
export const Table: Story = { render: framed(TABLE) }

/** The editor with what it holds beside it, so a change to the text is read back. */
const edited =
  (text: string): Render =>
  () => ({
    components: { Editor },
    setup: () => ({ held: ref(text) }),
    template: `
      <div class="h-screen bg-surface">
        <Editor v-model="held" class="h-2/3" />
        <pre data-source class="h-1/3 overflow-auto text-ink">{{ held }}</pre>
      </div>
    `,
  })

/** What the note now holds. */
const source = (canvas: HTMLElement) =>
  canvas.querySelector<HTMLElement>('[data-source]')?.textContent ?? ''

/** The cells of the one table drawn, head row first. */
const cells = (canvas: HTMLElement) =>
  [...canvas.querySelectorAll<HTMLElement>('.cm-table .cm-cell')]

/** A table with somewhere to put the caret that is not in it. */
const TABLED = `${TABLE}\nSomething well away from the table.\n`

/**
 * The table as a table. It is shown as it is written while the caret is in it,
 * so the caret is taken to the end of the note first.
 */
const asATable = async (canvas: HTMLElement) => {
  canvas.querySelector<HTMLElement>('.cm-content')?.focus()
  await userEvent.keyboard('{Control>}{End}{/Control}')
  await waitFor(() => expect(cells(canvas).length).toBeGreaterThan(0))
}

/**
 * A table typed into. The cells are the table, so what happens to one of them
 * is what happens to the markdown behind it.
 */
export const TableTypedInto: Story = {
  render: edited(TABLED),
  play: async ({ canvasElement }) => {
    await asATable(canvasElement)

    // The shape the markdown says, drawn as a table and not as three lines.
    const head = canvasElement.querySelectorAll('.cm-table thead .cm-cell')
    const body = canvasElement.querySelectorAll('.cm-table tbody tr')
    await expect(head).toHaveLength(3)
    await expect(body).toHaveLength(2)

    // A cell that is written somewhere is a cell that can be typed into.
    const first = cells(canvasElement)[0] as HTMLElement
    await expect(first.getAttribute('contenteditable')).toBe('plaintext-only')

    // Tab walks from one cell to the next rather than leaving the table.
    first.focus()
    await userEvent.tab()
    await expect(document.activeElement).toBe(cells(canvasElement)[1])

    // What is typed is written back over that cell alone. The browser does the
    // typing: the cell takes plain text, and only a real keystroke reaches it.
    const context = await import('vitest/browser')
    await context.userEvent.fill(cells(canvasElement)[3] as HTMLElement, 'Enthalpy')

    await waitFor(() => expect(source(canvasElement)).toContain('| Enthalpy | parent | 1865 |'))
    await expect(source(canvasElement)).toContain('| Temperature | child | 1848 |')
  },
}

/** A table grown by the buttons along the two edges it can grow along. */
export const TableGrown: Story = {
  render: edited(TABLED),
  play: async ({ canvasElement }) => {
    await asATable(canvasElement)
    const was = source(canvasElement)

    await userEvent.click(canvasElement.querySelector('.cm-add-column') as HTMLElement)
    await waitFor(() =>
      expect(canvasElement.querySelectorAll('.cm-table thead .cm-cell')).toHaveLength(4),
    )

    await userEvent.click(canvasElement.querySelector('.cm-add-row') as HTMLElement)
    await waitFor(() =>
      expect(canvasElement.querySelectorAll('.cm-table tbody tr')).toHaveLength(3),
    )

    // The markdown behind it grew with it, and what was in it is still there.
    await expect(source(canvasElement)).not.toBe(was)
    await expect(source(canvasElement)).toContain('Temperature')
  },
}

/* Three fenced blocks: two written in something the editor knows and one in a
   word no pack answers to. The word after the fence is the whole of what
   chooses a pack. */
const FENCED = [
  '```javascript',
  'const answered = 42',
  '```',
  '',
  '```python',
  'answered_here = 7',
  '```',
  '',
  '```notalanguage',
  'unpainted = 99',
  '```',
  '',
  'Something well away from all of it.',
  '',
].join('\n')

/** The line of a fenced block holding a run of text. */
const codeLine = (canvas: HTMLElement, holding: string) =>
  [...canvas.querySelectorAll<HTMLElement>('.cm-line')].find((line) =>
    line.textContent?.includes(holding),
  )

/** Everything on that line painted in something other than the line's own ink. */
const painted = (line: HTMLElement) => {
  const ink = getComputedStyle(line).color
  return [...line.querySelectorAll('span')].filter((span) => getComputedStyle(span).color !== ink)
}

/**
 * Fenced blocks in three words: two the editor has a pack for and one it has
 * none for. A pack is fetched after the block is drawn, so what it paints
 * arrives on a later frame.
 */
export const FencedInEveryLanguage: Story = {
  render: edited(FENCED),
  play: async ({ canvasElement }) => {
    const found = (holding: string) => {
      const line = codeLine(canvasElement, holding)
      expect(line, holding).toBeDefined()
      return line as HTMLElement
    }

    // The pack lands and paints the block it was fetched for.
    await waitFor(() => expect(painted(found('const answered')).length).toBeGreaterThan(0))
    await waitFor(() => expect(painted(found('answered_here')).length).toBeGreaterThan(0))

    // A word no pack answers to leaves the block plain, and leaves the text
    // itself alone.
    await expect(painted(found('unpainted'))).toHaveLength(0)
    await expect(source(canvasElement)).toContain('```notalanguage')
  },
}

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
const NOTE_TAB = 'note'
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
      const held = ref(pane('main', [NOTE_TAB, BESIDE], NOTE_TAB))
      const editors = new Map<string, { measure: () => void }>()
      return {
        held,
        NOTE_TAB,
        text: BEFORE,
        titles: { [NOTE_TAB]: 'Note', [BESIDE]: 'Beside' },
        choose: (tab: string) => {
          held.value = pane('main', [NOTE_TAB, BESIDE], tab)
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

    await userEvent.click(tab(NOTE_TAB))
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

const NESTED = '- a bullet\n'

/**
 * Tabbing in, and tabbing out again.
 *
 * Tab indents, which is what it does in every editor a list is written in, so
 * a person who tabbed into a note would be kept in it. Escape hands Tab back
 * to the page and the next Tab walks on; coming back arms it again, so every
 * visit begins the same way. The way out is read out as the keyboard arrives.
 */
export const LeftByTheKeyboard: Story = {
  render: () => ({
    components: { Editor },
    setup: () => ({ text: NESTED }),
    template: `
      <div class="numen flex h-screen flex-col bg-surface">
        <button
          data-before
          class="shrink-0 border-b border-rule px-3 py-2 text-left font-sans text-small text-ink"
        >
          Before
        </button>
        <Editor :model-value="text" class="min-h-0 flex-1" />
        <button
          data-after
          class="shrink-0 border-t border-rule px-3 py-2 text-left font-sans text-small text-ink"
        >
          After
        </button>
      </div>`,
  }),
  play: async ({ canvasElement }) => {
    const view = viewOf(canvasElement)
    const stop = (which: string) => canvasElement.querySelector(`[${which}]`) as HTMLElement

    // What a reader is told on arrival is the way back out.
    const told = document.getElementById(view.contentDOM.getAttribute('aria-describedby') ?? '')
    await expect(told?.textContent).toContain('Escape')

    stop('data-before').focus()
    await userEvent.tab()
    await expect(view.hasFocus).toBe(true)

    // Tab is the editor's while a person is writing: it indents and stays.
    await userEvent.tab()
    await expect(view.hasFocus).toBe(true)
    await expect(view.state.doc.toString()).not.toBe(NESTED)

    // Escape hands it to the page, and the next Tab is the page's.
    await userEvent.keyboard('{Escape}')
    await userEvent.tab()
    await expect(document.activeElement).toBe(stop('data-after'))

    // Back in, and Tab is the editor's again.
    await userEvent.tab({ shift: true })
    await expect(view.hasFocus).toBe(true)
    await userEvent.tab()
    await expect(view.hasFocus).toBe(true)
  },
}

/* A change something other than the reader is making. The editor draws it and
   never writes it: the button is what puts the text in, and until it is
   pressed the stretch about to be replaced is only marked. */
const NOTE =
  '# Entropy\n\n' +
  'A note is an ordinary file in an ordinary folder. Not a database and not ' +
  'an export: the files are the notes.\n'

/** Where a run of the text stands, as a change addresses it. */
const spanning = (text: string, run: string) => {
  const from = text.indexOf(run)
  return { from, to: from + run.length }
}

/** The text as it stands once the change has been made. */
const applied = (text: string, change: EditorChange) =>
  text.slice(0, change.from) + change.text + text.slice(change.to)

const making =
  (text: string, change: EditorChange): Render =>
  () => ({
    components: { Editor },
    setup: () => {
      const held = ref(text)
      return { held, change, make: () => (held.value = applied(text, change)) }
    },
    template: `
      <div class="numen flex h-screen flex-col bg-surface">
        <button
          class="shrink-0 border-b border-rule px-3 py-2 text-left font-sans text-small text-ink"
          @click="make"
        >
          Make the change
        </button>
        <Editor v-model="held" :change="change" class="min-h-0 flex-1" />
      </div>`,
  })

const MIDDLE: EditorChange = {
  id: 'middle',
  ...spanning(NOTE, 'an ordinary folder'),
  text: 'a folder anyone can open',
}

/**
 * A change partway through the text. The mark says where it is coming, and
 * the words arrive one at a time once the text has landed.
 */
export const Changed: Story = {
  render: making(NOTE, MIDDLE),
  play: async ({ canvasElement }) => {
    const view = viewOf(canvasElement)
    const arriving = () => canvasElement.querySelectorAll('.cm-arriving')

    await expect(canvasElement.querySelectorAll('.cm-changing').length).toBeGreaterThan(0)

    await userEvent.click(within(canvasElement).getByRole('button'))
    await settled()

    await expect(view.state.doc.toString()).toBe(applied(NOTE, MIDDLE))
    await expect(canvasElement.querySelector('.cm-changing')).toBeNull()
    await expect(view.contentDOM.textContent).not.toContain(MIDDLE.text)

    // One word arrives at a time: the same element for as long as it fades,
    // and the next word a run of its own. An element drawn again, or one
    // growing to hold everything shown so far, would fade a word twice.
    await waitFor(async () => await expect(arriving().length).toBe(1))
    const element = arriving()[0]
    const word = element?.textContent ?? ''
    await settled()
    await settled()
    await expect(arriving().length).toBe(1)
    await expect(arriving()[0]).toBe(element)

    await waitFor(async () => await expect(arriving()[0]?.textContent).not.toBe(word))
    await expect(arriving()[0]?.textContent?.startsWith(word)).toBe(false)

    await waitFor(async () => await expect(view.contentDOM.textContent).toContain(MIDDLE.text), {
      timeout: 5000,
    })
  },
}

/** A change at the very first character of the text. */
export const ChangedAtTheStart: Story = {
  render: making(NOTE, {
    id: 'start',
    ...spanning(NOTE, '# Entropy'),
    text: '# Entropy, and what it costs to keep',
  }),
}

const ENDING = NOTE.trimEnd()

/** A change running to the very last character of the text. */
export const ChangedAtTheEnd: Story = {
  render: making(ENDING, {
    id: 'end',
    ...spanning(ENDING, 'the files are the notes.'),
    text: 'the files are the notes, and the index is a cache.',
  }),
}

/** A change covering the whole of the text. */
export const ChangedThroughout: Story = {
  render: making(NOTE, {
    id: 'throughout',
    from: 0,
    to: NOTE.length,
    text: '# Entropy, rewritten\n\nEvery line of it is somebody else’s now.\n',
  }),
}

/** A change that puts nothing in: the stretch is marked and then it is gone. */
export const ChangedToNothing: Story = {
  render: making(NOTE, {
    id: 'nothing',
    ...spanning(NOTE, ' Not a database and not an export: the files are the notes.'),
    text: '',
  }),
}

/** A change far longer than what it replaces. */
export const ChangedForMore: Story = {
  render: making(NOTE, {
    id: 'more',
    ...spanning(NOTE, 'a database'),
    text:
      'a database, an index, a cache, or anything else that can be thrown ' +
      'away and made again from the files it was built out of',
  }),
}

const SCRIPTS = `# ${RUSSIAN}\n\n${ARABIC}\n\n${DEVANAGARI}\n`

/** A change written in a script that is not Latin, over one that runs the
 *  other way. */
export const ChangedInAnotherScript: Story = {
  render: making(SCRIPTS, { id: 'script', ...spanning(SCRIPTS, ARABIC), text: DEVANAGARI }),
}

const SETTINGS = `{
  // The window
  "appearance": { "theme": "preset:numen", "text_scale": 1 },
  "indexing": {
    "embedding": { "model": { "name": "held/tiny-e5-small" } },
    "transcribe_recordings": true,
    "transcribe_under_mb": 200
  },
  "agent": { "use": "claude", "serve_tools": false }
}
`

/**
 * A whole document written in one language: read as that language, and set in
 * the face code is set in. This is what a settings file is opened in.
 */
export const AWholeDocumentOfCode: Story = {
  render: framed(SETTINGS, { live: false, language: 'json' }),
}

/** A language no fence answers to leaves the document plain. */
export const ALanguageNothingAnswersTo: Story = {
  render: framed(SETTINGS, { live: false, language: 'not-a-language' }),
}
