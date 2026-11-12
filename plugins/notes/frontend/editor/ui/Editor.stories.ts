/**
 * What the editor looks like while it is being typed into. What it does to the
 * text is asserted in `live.test.ts`, `table.test.ts` and `change.test.ts`.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { createApp, ref } from 'vue'
import { autocompletion, startCompletion } from '@codemirror/autocomplete'
import { undo } from '@codemirror/commands'
import { EditorSelection } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import Editor from './Editor.vue'
import type { EditorChange } from '../lib/change/change'
import { MARKED_UP, TABLE } from '../fixtures/markdown'
import { ARABIC, DEVANAGARI, LINK, LONG, RUSSIAN, UNBREAKABLE } from '../fixtures/prose'

const meta = {
  title: 'Notes/Editor',
  component: Editor,
  parameters: {
    layout: 'fullscreen',
    // Tab indents while the editor has focus; Escape returns Tab to the page.
    // The announcement on focus names that exit.
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

const renderEditor =
  (text: string, props: Record<string, unknown> = {}): Render =>
  () => ({
    components: { Editor },
    setup: () => ({ text, props }),
    template: `<Editor :model-value="text" v-bind="props" class="h-screen bg-surface" />`,
  })

/** One of everything: headings, marks, lists, a rule, a table, code and maths. */
export const Playground: Story = { render: renderEditor(MARKED_UP) }

/** The marks as they are written, with nothing drawn for them. */
export const AsWritten: Story = { render: renderEditor(MARKED_UP, { isLive: false }) }

/** Nothing to type into. */
export const ReadOnly: Story = { render: renderEditor(MARKED_UP, { isReadonly: true }) }

/* One line of every construct whose marks are concealed, each with a blank
   line after it, and a line at the end to park the caret on. */
const BULLET = '- a bullet'
const TASK = '- [ ] something to do'
const RULE = '---'
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
  RULE,
]

const EVERY = CONCEALED
const STEADY = `${EVERY.join('\n\n')}\n\nSomething well away from all of it.\n`

const getLineOf = (text: string) => EVERY.indexOf(text) * 2 + 1
const PARKED = EVERY.length * 2 + 1

/* A heading is set at the tightest line in the editor, and how far a letter
   is above and below its baseline is the font's answer. These are the
   machine's own sans cut twice: one face reaching past that line, one well
   inside it. */
const SANS =
  "local('DejaVu Sans'), local('Liberation Sans'), local('Noto Sans'), " +
  "local('FreeSans'), local('Nimbus Sans'), local('Arial'), local('Helvetica'), " +
  "local('Cantarell'), local('Ubuntu'), local('Roboto'), local('Verdana')"
const TALL = { name: 'numen-tall', size: '150%' }
const LOW = { name: 'numen-low', size: '60%' }
const FACES = [TALL, LOW]

const loadFaces = () => {
  const sheet = document.createElement('style')
  sheet.textContent = FACES.map(
    ({ name, size }) => `@font-face { font-family: '${name}'; src: ${SANS}; size-adjust: ${size} }`,
  ).join('\n')
  document.head.append(sheet)
  return Promise.all(FACES.map(({ name }) => document.fonts.load(`100px '${name}'`)))
}

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
 * The marks of a construct are concealed until the caret comes to its line. A rule is drawn until
 * the caret is on it, and a click at the start of a line lands on that line. Every check is made
 * under the machine's font and under both cut faces.
 */
export const ClickLandsOnTheLine: Story = {
  render: renderEditor(STEADY),
  play: async ({ canvasElement }) => {
    const view = getView(canvasElement)

    const setCaret = async (line: number, column: number) => {
      view.dispatch({ selection: EditorSelection.single(view.state.doc.line(line).from + column) })
      await nextFrame()
      await nextFrame()
    }

    await loadFaces()
    // A face the machine cannot cut pins nothing.
    await expect(reach(TALL.name) > reach(LOW.name)).toBe(true)

    for (const selected of [null, ...FACES]) {
      const face = selected ? selected.name : 'the machine'
      if (selected) view.dom.style.setProperty('--numen-font-sans', `'${selected.name}'`)
      else view.dom.style.removeProperty('--numen-font-sans')
      await nextFrame()

      // The rule is drawn until the caret is on it.
      await setCaret(PARKED, 0)
      await expect(view.contentDOM.querySelectorAll('.cm-rule')).toHaveLength(1)
      await setCaret(getLineOf(RULE), 1)
      await expect(view.contentDOM.querySelectorAll('.cm-rule')).toHaveLength(0)

      // A caret at the start of a line is drawn there, and a click there lands on that line.
      await setCaret(PARKED, 0)
      for (const [text, column] of [
        ['# Heading one', 0],
        ['# Heading one', 2],
        [BULLET, 0],
        [BULLET, 1],
        [TASK, 0],
        [TASK, 1],
      ] as const) {
        const line = getLineOf(text)
        const spot = view.coordsAtPos(view.state.doc.line(line).from + column)
        const back = spot
          ? view.posAtCoords({ x: spot.left + 1, y: (spot.top + spot.bottom) / 2 })
          : null
        await expect({
          face,
          text,
          column,
          drawn: !!spot,
          lands: back !== null && view.state.doc.lineAt(back).number === line,
        }).toEqual({ face, text, column, drawn: true, lands: true })
      }
    }
  },
}

/**
 * A table in isolation. Typing in a cell writes back to that cell only. Tab
 * moves between cells and adds a row at the end, Escape leaves the table, and
 * the buttons on its edges add a row or a column.
 */
export const Table: Story = { render: renderEditor(TABLE) }

const renderWithSource =
  (text: string): Render =>
  () => ({
    components: { Editor },
    setup: () => ({ source: ref(text) }),
    template: `
      <div class="h-screen bg-surface">
        <Editor v-model="source" class="h-2/3" />
        <pre data-source class="h-1/3 overflow-auto text-ink">{{ source }}</pre>
      </div>
    `,
  })

/** The key CodeMirror reads as Mod: Command on a Mac, Control elsewhere. */
const MOD = /Mac/.test(navigator.platform) ? 'Meta' : 'Control'

const source = (canvas: HTMLElement) =>
  canvas.querySelector<HTMLElement>('[data-source]')?.textContent ?? ''

const cells = (canvas: HTMLElement) => [
  ...canvas.querySelectorAll<HTMLElement>('.cm-table .cm-cell'),
]

/** A table with somewhere to put the caret that is not in it. */
const TABLED = `${TABLE}\nSomething well away from the table.\n`

const asATable = async (canvas: HTMLElement) => {
  canvas.querySelector<HTMLElement>('.cm-content')?.focus()
  await userEvent.keyboard(`{${MOD}>}{End}{/${MOD}}`)
  await waitFor(() => expect(cells(canvas).length).toBeGreaterThan(0))
}

/**
 * A table typed into. The cells are the table, so what happens to one of them
 * is what happens to the markdown behind it.
 */
export const TableTypedInto: Story = {
  render: renderWithSource(TABLED),
  play: async ({ canvasElement }) => {
    await asATable(canvasElement)

    // The markdown renders as a table, not as three lines.
    const head = canvasElement.querySelectorAll('.cm-table thead .cm-cell')
    const body = canvasElement.querySelectorAll('.cm-table tbody tr')
    await expect(head).toHaveLength(3)
    await expect(body).toHaveLength(2)

    // A cell that is written somewhere is a cell that can be typed into.
    const first = cells(canvasElement)[0] as HTMLElement
    await expect(first.getAttribute('contenteditable')).toBe('plaintext-only')

    // Tab walks from one cell to the next and not leaving the table.
    first.focus()
    await userEvent.tab()
    await expect(document.activeElement).toBe(cells(canvasElement)[1])

    // Typing writes back to that cell only. The browser performs the typing: the
    // cell takes text input only from a real keystroke.
    const context = await import('vitest/browser')
    await context.userEvent.fill(cells(canvasElement)[3] as HTMLElement, 'Enthalpy')

    await waitFor(() => expect(source(canvasElement)).toContain('| Enthalpy | parent | 1865 |'))
    await expect(source(canvasElement)).toContain('| Temperature | child | 1848 |')
  },
}

/** A table grown by the buttons along the two edges it can grow along. */
export const TableGrown: Story = {
  render: renderWithSource(TABLED),
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

/*
 * Three fenced blocks: two in languages the editor supports and one in a
 * language no pack matches. The word after the fence selects the pack.
 */
const FENCED = [
  '```javascript',
  'const answerVal = 42',
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

const codeLine = (canvas: HTMLElement, text: string) =>
  [...canvas.querySelectorAll<HTMLElement>('.cm-line')].find((line) =>
    line.textContent?.includes(text),
  )

const getHighlightSpans = (line: HTMLElement) => [...line.querySelectorAll('span')]

/**
 * Fenced blocks in three words: two the editor has a pack for and one it has
 * none for. A pack is fetched after the block is drawn, so what it paints
 * arrives on a later frame.
 */
export const FencedInEveryLanguage: Story = {
  render: renderWithSource(FENCED),
  play: async ({ canvasElement }) => {
    const found = (text: string) => {
      const line = codeLine(canvasElement, text)
      if (!line) throw new Error(`no code line holds ${text}`)
      return line
    }

    // The pack lands and paints the block it was fetched for.
    await waitFor(() =>
      expect(getHighlightSpans(found('const answerVal')).length).toBeGreaterThan(0),
    )
    await waitFor(() => expect(getHighlightSpans(found('answered_here')).length).toBeGreaterThan(0))

    // A word no pack answers to leaves the block plain, and leaves the text
    // itself alone.
    await expect(getHighlightSpans(found('unpainted'))).toHaveLength(0)
    await expect(source(canvasElement)).toContain('```notalanguage')
  },
}

/** Nothing written yet, and something to say so. */
export const Empty: Story = { render: renderEditor('', { placeholder: 'Write' }) }

/** Empty text with an empty placeholder. */
export const NothingAtAll: Story = { render: renderEditor('', { placeholder: '' }) }

/** One line, which is what a note is for its first minute. */
export const OneLine: Story = { render: renderEditor('# Entropy\n') }

/** Far too many lines. What the scrolling is for. */
export const FarTooMany: Story = {
  render: renderEditor(
    Array.from({ length: 400 }, (_, index) =>
      index % 8 === 0 ? `## Part ${index / 8 + 1}` : `${index}. ${RUSSIAN}`,
    ).join('\n'),
  ),
}

/** One paragraph past any width, and a quotation of the same. */
export const FarTooLong: Story = {
  render: renderEditor(`${LONG} ${LONG} ${LONG}\n\n> ${LONG}\n`),
}

/** A word and an address with nowhere in them to break. */
export const NothingToBreakAt: Story = {
  render: renderEditor(`${UNBREAKABLE}\n\n- ${UNBREAKABLE}\n- <${LINK}>\n\n\`${UNBREAKABLE}\`\n`),
}

/** Scripts that are not Latin, one that runs the other way, and a word with
 *  nowhere to break. */
export const AwkwardText: Story = {
  render: renderEditor(
    `# ${RUSSIAN}\n\n${DEVANAGARI}\n\n${ARABIC}\n\n- ${UNBREAKABLE}\n- <${LINK}>\n`,
  ),
}

const BODY = Array.from({ length: 120 }, (_, index) => `${index + 1}. ${RUSSIAN}`).join('\n')
const BEFORE = `# Entropy\n\n${BODY}\n`
const AFTER = `# Entropy, retitled by whoever else has this open\n\n${BODY}\n`

/** Where the caret was put before the text was read again. */
const LINE = 40
const COLUMN = 3

const getView = (canvas: HTMLElement): EditorView => {
  const view = EditorView.findFromDOM(canvas.querySelector('.editor') as HTMLElement)
  if (!view) throw new Error('the editor drew no view')
  return view
}

const nextFrame = () => new Promise((isDone) => requestAnimationFrame(() => isDone(null)))

const topmost = (view: EditorView): number => {
  const box = view.scrollDOM.getBoundingClientRect()
  const pos = view.posAtCoords({ x: box.left + 8, y: box.top + 8 }) ?? 0
  return view.state.doc.lineAt(pos).number
}

/**
 * The text reloaded after an external change to the same file while it is
 * open. The caret stays on its line, the scroll offset is unchanged, and undo
 * does not restore the old text.
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
    const view = getView(canvasElement)

    view.dispatch({ selection: EditorSelection.single(view.state.doc.line(LINE).from + COLUMN) })
    view.scrollDOM.scrollTop = 600
    await nextFrame()
    await expect(view.scrollDOM.scrollTop).toBeGreaterThan(0)
    const top = topmost(view)

    await userEvent.click(within(canvasElement).getByRole('button'))
    await nextFrame()
    await nextFrame()

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

    await nextFrame()
    const room = canvasElement.querySelector('[data-tabs]') as HTMLElement
    await expect(room.querySelectorAll('.cm-editor').length).toBe(1)
    const one = heap()

    // Nine more of what a pane holds out of sight, in the page that already
    // has one, so the difference is the tabs and not the page.
    const more = Array.from({ length: 9 }, () => {
      const element = document.createElement('div')
      element.style.display = 'none'
      room.append(element)
      const app = createApp(Editor, { modelValue: MARKED_UP })
      app.mount(element)
      return app
    })
    await nextFrame()
    await nextFrame()
    await expect(room.querySelectorAll('.cm-editor').length).toBe(10)
    const ten = heap()

    console.info(
      `one open tab: ${mb(one)} MB; ten: ${mb(ten)} MB; each further: ${mb((ten - one) / 9)} MB`,
    )
    more.forEach((app) => app.unmount())
  },
}

/**
 * The save chord.
 *
 * Put the caret in the text and press Ctrl+S. The editor emits a save request,
 * leaves the text unchanged, and the page's own handler for the chord does not
 * run.
 */
export const Kept: Story = {
  render: () => ({
    components: { Editor },
    setup: () => {
      const request = ref(0)
      const reply = ref('')
      return {
        text: MARKED_UP,
        saveCount: request,
        outcome: reply,
        onSave: () => (request.value += 1),
        watch: (key: KeyboardEvent) => {
          if (key.key === 's' && (key.ctrlKey || key.metaKey)) {
            reply.value = key.defaultPrevented ? 'prevented' : 'passed'
          }
        },
      }
    },
    template: `
      <div class="numen flex h-screen flex-col bg-surface" @keydown="watch">
        <p class="shrink-0 border-b border-rule px-3 py-2 font-sans text-small text-ink">
          saved <span data-save-count>{{ saveCount }}</span> times, and
          <span data-outcome>{{ outcome }}</span>
        </p>
        <Editor :model-value="text" class="min-h-0 flex-1" @save="onSave" />
      </div>`,
  }),
  play: async ({ canvasElement }) => {
    const view = getView(canvasElement)
    const getText = (what: string) =>
      (canvasElement.querySelector(`[${what}]`) as HTMLElement).textContent?.trim()

    await userEvent.click(view.contentDOM)
    await nextFrame()
    await expect(view.hasFocus).toBe(true)

    await userEvent.keyboard(`{${MOD}>}s{/${MOD}}`)
    await nextFrame()

    await expect(getText('data-save-count')).toBe('1')
    await expect(getText('data-outcome')).toBe('prevented')
    await expect(view.state.doc.toString()).toBe(MARKED_UP)
  },
}

const NESTED = '- a bullet\n'

/**
 * Tab in and Tab out.
 * Tab indents, so focus would stay inside the note. Escape returns Tab to the page and the next Tab moves on.
 * Re-entering the editor re-arms the trap. The exit is announced when the keyboard arrives.
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
    const view = getView(canvasElement)
    const stop = (which: string) => canvasElement.querySelector(`[${which}]`) as HTMLElement

    // The announcement on arrival names the exit.
    const notifiedElement = document.getElementById(
      view.contentDOM.getAttribute('aria-describedby') ?? '',
    )
    await expect(notifiedElement?.textContent).toContain('Escape')

    stop('data-before').focus()
    await userEvent.tab()
    await expect(view.hasFocus).toBe(true)

    // While writing, Tab belongs to the editor: it indents and keeps focus.
    await userEvent.tab()
    await expect(view.hasFocus).toBe(true)
    await expect(view.state.doc.toString()).not.toBe(NESTED)

    // Escape releases Tab to the page, and the next Tab moves focus on.
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

const findSpan = (text: string, run: string) => {
  const from = text.indexOf(run)
  return { from, to: from + run.length }
}

const applyChange = (text: string, change: EditorChange) =>
  text.slice(0, change.from) + change.text + text.slice(change.to)

const renderWithChange =
  (text: string, change: EditorChange): Render =>
  () => ({
    components: { Editor },
    setup: () => {
      const target = ref(text)
      return { source: target, change, make: () => (target.value = applyChange(text, change)) }
    },
    template: `
      <div class="numen flex h-screen flex-col bg-surface">
        <button
          class="shrink-0 border-b border-rule px-3 py-2 text-left font-sans text-small text-ink"
          @click="make"
        >
          Make the change
        </button>
        <Editor v-model="source" :change="change" class="min-h-0 flex-1" />
      </div>`,
  })

const MIDDLE: EditorChange = {
  id: 'middle',
  ...findSpan(NOTE, 'an ordinary folder'),
  text: 'a folder anyone can open',
}

/**
 * A change in the middle of the text. The mark shows where it will land, and
 * the words arrive one at a time once the text is in place.
 */
export const Changed: Story = {
  render: renderWithChange(NOTE, MIDDLE),
  play: async ({ canvasElement }) => {
    const view = getView(canvasElement)
    const getArrivingElements = () => canvasElement.querySelectorAll('.cm-arriving')

    await expect(canvasElement.querySelectorAll('.cm-changing').length).toBeGreaterThan(0)

    await userEvent.click(within(canvasElement).getByRole('button'))
    await nextFrame()

    await expect(view.state.doc.toString()).toBe(applyChange(NOTE, MIDDLE))
    await expect(canvasElement.querySelector('.cm-changing')).toBeNull()
    await expect(view.contentDOM.textContent).not.toContain(MIDDLE.text)

    // One word arrives at a time: the same element for as long as it fades,
    // and the next word a run of its own. An element drawn again, or one
    // growing to hold everything shown so far, would fade a word twice.
    await waitFor(async () => await expect(getArrivingElements().length).toBe(1))
    const element = getArrivingElements()[0]
    const word = element?.textContent ?? ''
    await nextFrame()
    await nextFrame()
    await expect(getArrivingElements().length).toBe(1)
    await expect(getArrivingElements()[0]).toBe(element)

    await waitFor(async () => await expect(getArrivingElements()[0]?.textContent).not.toBe(word))
    await expect(getArrivingElements()[0]?.textContent?.startsWith(word)).toBe(false)

    await waitFor(async () => await expect(view.contentDOM.textContent).toContain(MIDDLE.text), {
      timeout: 5000,
    })
  },
}

/** A change at the first character of the text. */
export const ChangedAtTheStart: Story = {
  render: renderWithChange(NOTE, {
    id: 'start',
    ...findSpan(NOTE, '# Entropy'),
    text: '# Entropy, and what it costs to keep',
  }),
}

const ENDING = NOTE.trimEnd()

/** A change that extends to the last character of the text. */
export const ChangedAtTheEnd: Story = {
  render: renderWithChange(ENDING, {
    id: 'end',
    ...findSpan(ENDING, 'the files are the notes.'),
    text: 'the files are the notes, and the index is a cache.',
  }),
}

/** A change covering the whole of the text. */
export const ChangedThroughout: Story = {
  render: renderWithChange(NOTE, {
    id: 'throughout',
    from: 0,
    to: NOTE.length,
    text: '# Entropy, rewritten\n\nEvery line of it is somebody else’s now.\n',
  }),
}

/** A change that puts nothing in: the stretch is marked and then it is gone. */
export const ChangedToNothing: Story = {
  render: renderWithChange(NOTE, {
    id: 'nothing',
    ...findSpan(NOTE, ' Not a database and not an export: the files are the notes.'),
    text: '',
  }),
}

/** A change far longer than what it replaces. */
export const ChangedForMore: Story = {
  render: renderWithChange(NOTE, {
    id: 'more',
    ...findSpan(NOTE, 'a database'),
    text:
      'a database, an index, a cache, or anything else that can be thrown ' +
      'away and made again from the files it was built out of',
  }),
}

const SCRIPTS = `# ${RUSSIAN}\n\n${ARABIC}\n\n${DEVANAGARI}\n`

/** A change written in a script that is not Latin, over one that runs the
 *  other way. */
export const ChangedInAnotherScript: Story = {
  render: renderWithChange(SCRIPTS, {
    id: 'script',
    ...findSpan(SCRIPTS, ARABIC),
    text: DEVANAGARI,
  }),
}

const SETTINGS = `{
  // The window
  "app": {
    "appearance": { "theme": "preset:numen", "textScale": 1 },
    "agent": { "use": "claude", "serve_tools": false }
  },
  "md.numen.engines.embeddings": { "model": "held/tiny-e5-small" }
}
`

/**
 * A whole document written in one language: read as that language, and set in
 * the face code is set in. This is what a settings file is opened in.
 */
export const AWholeDocumentOfCode: Story = {
  render: renderEditor(SETTINGS, { isLive: false, language: 'json' }),
}

/** A language no fence answers to leaves the document plain. */
export const ALanguageNothingAnswersTo: Story = {
  render: renderEditor(SETTINGS, { isLive: false, language: 'not-a-language' }),
}

/* ------------------------------- Wikilinks -------------------------------- */

/** One of each form of a wikilink on its own line, then a line to park the caret on. */
const WIKILINKS =
  '[[Entropy]]\n\n[[Entropy|disorder]]\n\n[[Entropy#Heat]]\n\nA [[Carnot]] and [[Kelvin]] side by side.\n\n[[unfinished\n\nelsewhere\n'

const playWikilinksAway = async (canvasElement: HTMLElement) => {
  const view = getView(canvasElement)
  view.dispatch({ selection: EditorSelection.single(view.state.doc.length) })
  await nextFrame()

  const lines = Array.from(view.contentDOM.querySelectorAll('.cm-line')).map(
    (one) => one.textContent,
  )
  await expect(lines).toContain('Entropy')
  await expect(lines).toContain('disorder')
  await expect(lines).toContain('Entropy#Heat')
  await expect(lines).toContain('A Carnot and Kelvin side by side.')
  await expect(lines).toContain('[[unfinished')

  const links = Array.from(view.contentDOM.querySelectorAll('.cm-wikilink'))
  await expect(links.map((one) => one.textContent)).toEqual([
    'Entropy',
    'disorder',
    'Entropy#Heat',
    'Carnot',
    'Kelvin',
  ])
  await expect(view.contentDOM.querySelectorAll('.cm-wikilink-mark')).toHaveLength(0)
  await expect(view.contentDOM.querySelectorAll('.cm-link')).toHaveLength(0)
}

/** Away from the caret a wikilink shows its target, or its alias, as one link and nothing else. */
export const WikilinksAway: Story = {
  render: renderEditor(WIKILINKS),
  play: ({ canvasElement }) => playWikilinksAway(canvasElement),
}

const playWikilinkTouched = async (canvasElement: HTMLElement) => {
  const view = getView(canvasElement)
  for (const caret of [0, 5, 11]) {
    view.dispatch({ selection: EditorSelection.single(caret) })
    await nextFrame()

    const first = view.contentDOM.querySelector('.cm-line') as HTMLElement
    await expect(first.textContent).toBe('[[Entropy]]')
    const brackets = Array.from(first.querySelectorAll('.cm-wikilink-mark'))
    await expect(brackets.map((one) => one.textContent)).toEqual(['[[', ']]'])
    await expect(first.querySelector('.cm-wikilink')).not.toBeNull()
    await expect(view.contentDOM.querySelectorAll('.cm-matchingBracket')).toHaveLength(0)
    await expect(view.contentDOM.querySelectorAll('.cm-nonmatchingBracket')).toHaveLength(0)
  }
}

/** With the caret in or touching the link its brackets are back, both muted alike. */
export const WikilinkTouched: Story = {
  render: renderEditor(WIKILINKS),
  play: ({ canvasElement }) => playWikilinkTouched(canvasElement),
}

/* ---------------------------- The completion list --------------------------- */

const completionOf = autocompletion({
  override: [
    (context) => ({
      from: context.pos - 2,
      filter: false,
      options: [
        { label: 'Entropy', detail: 'note', type: 'class' },
        { label: 'Entropy and heat', detail: 'note', type: 'class' },
      ],
    }),
  ],
})

const playCompletionOpened = async (canvasElement: HTMLElement) => {
  const view = getView(canvasElement)
  view.focus()
  view.dispatch({ selection: EditorSelection.single(view.state.doc.length) })
  startCompletion(view)

  const list = await waitFor(() => {
    const found = canvasElement.ownerDocument.querySelector('.cm-tooltip-autocomplete')
    if (!found) throw new Error('the completion list is not open')
    return found
  })
  await expect([...list.querySelectorAll('li')].map((one) => one.textContent)).toEqual([
    'Entropynote',
    'Entropy and heatnote',
  ])
  await expect(list.querySelectorAll('li[aria-selected]')).toHaveLength(1)
}

/** The completion list opens with the matching notes, one of them selected. */
export const CompletionOpen: Story = {
  render: renderEditor('See [[En', { extensions: completionOf }),
  play: ({ canvasElement }) => playCompletionOpened(canvasElement),
}
