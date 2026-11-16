/**
 * The query edited in the code editor inside the tab of a saved search note, shown and hidden by the
 * edit button: completion from the catalog, the query saved once typing pauses, and the problems of the
 * saved query and of a query not saved underlined with their message on hover.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import { FAILED_ON_RERANKER, MISSPELLED, NOTE_PATH, RERANKER_BUSY } from '../fixtures/responses'
import { getDot, renderTab, TAB_ARGS, type Knobs } from '../fixtures/tabStory'
import { QUERY_PAUSE_MS } from '../model/queryPause'

const meta: Meta<Knobs> = {
  title: 'Saved search/Query editor',
  parameters: { layout: 'fullscreen' },
  args: TAB_ARGS,
  render: renderTab,
}

export default meta
type Story = StoryObj<Knobs>

const NEW_QUERY = 'words("lighthouse") && meaning("the keeper in a gale").top(20)'

const getEditor = (canvasElement: HTMLElement): HTMLElement =>
  within(canvasElement).getByRole('textbox', { name: 'Query' })

const openEditor = async (canvasElement: HTMLElement): Promise<HTMLElement> => {
  const canvas = within(canvasElement)
  await userEvent.click(await waitFor(() => canvas.getByRole('button', { name: 'Edit query' })))
  await waitFor(() => expect(canvas.getByRole('region', { name: 'Edit query' })).toBeVisible())
  return getEditor(canvasElement)
}

const fillEditor = async (editor: HTMLElement, text: string): Promise<void> => {
  const browser = await import('vitest/browser')
  await browser.userEvent.fill(editor, text)
}

const AFTER_PAUSE = { timeout: QUERY_PAUSE_MS * 3 }

/** Tab inserts indentation while the query editor has focus; Escape hands Tab back to the page. */
const EDITOR_KEEPS_TAB = { keeps: 'Escape hands Tab back to the page' }

/** A run that failed on a busy reranking service: the red dot gives its error, nothing is underlined. */
export const ShowsTheErrorOfTheRunApartFromTheQuery: Story = {
  parameters: { reach: EDITOR_KEEPS_TAB },
  args: { response: FAILED_ON_RERANKER },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const dot = await getDot(canvasElement, 'Failed')

    await expect(canvas.queryByText(RERANKER_BUSY)).toBeNull()
    await userEvent.hover(dot)
    await waitFor(() => expect(canvas.getByRole('tooltip')).toHaveTextContent(RERANKER_BUSY))

    await openEditor(canvasElement)
    await expect(canvasElement.querySelectorAll('.cm-lintRange-error')).toHaveLength(0)
  },
}

/** The edit button shows the query in the editor inside the tab, and hides it again. */
export const ShowsAndHidesTheEditor: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const button = await waitFor(() => canvas.getByRole('button', { name: 'Edit query' }))
    await expect(button).toHaveAttribute('aria-expanded', 'false')

    const editor = await openEditor(canvasElement)

    await expect(button).toHaveAttribute('aria-expanded', 'true')
    await expect(editor).toHaveTextContent('meaning("the harbor master in a winter storm")')
    await expect(canvas.getByRole('button', { name: /1911/ })).toBeVisible()
    await expect(canvas.queryByRole('dialog')).toBeNull()

    await userEvent.click(button)

    await waitFor(() => expect(canvas.queryByRole('region', { name: 'Edit query' })).toBeNull())
    await expect(button).toHaveAttribute('aria-expanded', 'false')
  },
}

/** A valid query is saved once typing pauses, the results are read again, and the editor stays open. */
export const SavesTheQueryAfterTypingPauses: Story = {
  parameters: { reach: EDITOR_KEEPS_TAB },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const editor = await openEditor(canvasElement)
    const readsBefore = args.onGetResults.mock.calls.length

    await fillEditor(editor, NEW_QUERY)

    await expect(args.onUpdateSearch).not.toHaveBeenCalled()
    await waitFor(
      () => expect(args.onUpdateSearch).toHaveBeenCalledWith({ path: NOTE_PATH, query: NEW_QUERY }),
      AFTER_PAUSE,
    )
    await expect(args.onUpdateSearch).toHaveBeenCalledTimes(1)
    await waitFor(() => expect(args.onGetResults.mock.calls.length).toBeGreaterThan(readsBefore))
    await expect(canvas.getByRole('region', { name: 'Edit query' })).toBeVisible()
    await expect(canvasElement.querySelectorAll('.cm-lintRange-error')).toHaveLength(0)

    await userEvent.click(canvas.getByRole('button', { name: 'Edit query' }))
    await expect(await openEditor(canvasElement)).toHaveTextContent(NEW_QUERY)
  },
}

/**
 * Typing a name offers the functions of the catalog, each with what it does in plain words and its
 * description in its row, in a list drawn outside the tab so the tab does not cut it; a click on one
 * writes it; after a dot, the stages are offered.
 */
export const CompletesTheNamesOfTheCatalog: Story = {
  parameters: { reach: EDITOR_KEEPS_TAB },
  play: async ({ canvasElement }) => {
    const page = within(document.body)
    const browser = await import('vitest/browser')
    const editor = await openEditor(canvasElement)

    await fillEditor(editor, 'words("gale") && t')
    await browser.userEvent.keyboard('a')

    const list = await waitFor(() => page.getByRole('listbox'))
    await waitFor(() =>
      expect(
        within(list)
          .getAllByRole('option')
          .map((one) => one.textContent),
      ).toStrictEqual([expect.stringContaining('notes with this tag')]),
    )
    await expect(canvasElement.querySelector('.saved-search')?.contains(list)).toBe(false)
    await expect(within(list).getByText(/Finds notes that carry the tag you give\./)).toBeVisible()
    await expect(within(list).getByText(/tags\("harbor"\)/)).toBeVisible()
    await expect(document.querySelector('.cm-completionInfo')).toBeNull()

    await browser.userEvent.click(within(list).getByRole('option'))
    await waitFor(() => expect(editor).toHaveTextContent('words("gale") && tags'))

    await browser.userEvent.keyboard('("harbor").r')
    const stages = await waitFor(() => page.getByRole('listbox'))
    await waitFor(() =>
      expect(
        within(stages)
          .getAllByRole('option')
          .map((one) => one.textContent),
      ).toStrictEqual([expect.stringContaining('rerank')]),
    )
  },
}

/** A saved query the core rejects is underlined at its place as soon as the editor opens. */
export const UnderlinesTheProblemOfTheSavedQuery: Story = {
  parameters: { reach: EDITOR_KEEPS_TAB },
  args: { response: MISSPELLED },
  play: async ({ canvasElement }) => {
    await openEditor(canvasElement)

    const underlined = await waitFor(() => {
      const marks = canvasElement.querySelectorAll('.cm-lintRange-error')
      if (marks.length !== 1) throw new Error(`${marks.length} underlined ranges, want one`)
      return marks[0]!
    })
    await expect(underlined).toHaveTextContent(/^meanin$/)
  },
}

const PROBLEM = 'Unknown function meanin'

/**
 * A query the core rejects is underlined at its place with the core's message in one tooltip on hover,
 * drawn outside the tab, and listed as line:column and message below the query, where a click puts the
 * cursor at it; the note keeps its query, and the editor stays open.
 */
export const UnderlinesAQueryNotSaved: Story = {
  parameters: { reach: EDITOR_KEEPS_TAB },
  args: {
    updateAnswer: { errors: [{ line: 1, column: 18, message: PROBLEM }] },
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const editor = await openEditor(canvasElement)
    const readsBefore = args.onGetResults.mock.calls.length

    await fillEditor(editor, 'words("gale") && meanin("a night at sea")')

    await waitFor(() => expect(args.onUpdateSearch).toHaveBeenCalledTimes(1), AFTER_PAUSE)
    const underlined = await waitFor(() => {
      const marks = canvasElement.querySelectorAll('.cm-lintRange-error')
      if (marks.length !== 1) throw new Error(`${marks.length} underlined ranges, want one`)
      return marks[0]!
    })
    await expect(underlined).toHaveTextContent(/^meanin$/)
    await expect(canvas.queryByRole('alert')).toBeNull()

    const browser = await import('vitest/browser')
    await browser.userEvent.hover(underlined)
    const lint = await waitFor(() => {
      const shown = document.querySelectorAll('.cm-tooltip-lint')
      if (shown.length !== 1) throw new Error(`${shown.length} problem tooltips, want one`)
      return shown[0]!
    })
    await expect(lint).toHaveTextContent(PROBLEM)
    await expect(canvasElement.querySelector('.saved-search')?.contains(lint)).toBe(false)

    const problems = canvas.getByRole('list', { name: 'Problems in the query' })
    await expect(problems).toHaveTextContent(new RegExp(`^\\s*1:18\\s*${PROBLEM}\\s*$`))
    await browser.userEvent.click(within(problems).getByRole('button'))
    await browser.userEvent.keyboard('X')
    await expect(editor).toHaveTextContent('words("gale") && Xmeanin("a night at sea")')

    await expect(canvas.getByRole('region', { name: 'Edit query' })).toBeVisible()
    await expect(args.onGetResults.mock.calls.length).toBe(readsBefore)

    await userEvent.click(canvas.getByRole('button', { name: 'Edit query' }))
    await expect(await openEditor(canvasElement)).toHaveTextContent(
      'meaning("the harbor master in a winter storm")',
    )
  },
}

/** Tab inserts indentation at the cursor and leaves the rest of the line where it is. */
export const InsertsIndentationAtTheCursor: Story = {
  parameters: { reach: EDITOR_KEEPS_TAB },
  play: async ({ canvasElement }) => {
    const browser = await import('vitest/browser')
    const editor = await openEditor(canvasElement)

    await fillEditor(editor, 'words("gale") && tags("sea")')
    await browser.userEvent.keyboard('{ArrowLeft}'.repeat('tags("sea")'.length))
    await browser.userEvent.keyboard('{Tab}')

    await waitFor(() => expect(editor.textContent).toBe('words("gale") &&   tags("sea")'))
    await expect(editor).toHaveFocus()
  },
}
