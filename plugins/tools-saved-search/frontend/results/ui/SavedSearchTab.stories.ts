/**
 * The tab of a saved search note over a scripted service: the query and its results, a passage and a
 * whole file opened, the dot of the state of the run and its tooltip, the problems of the query with the
 * last good results, no result, a warning, a failed last run, the filter and the sources of the results
 * and the button that brings back what they hide, and the query edited in the code editor inside the
 * tab, shown and hidden by the edit button: completion from the catalog, the query saved once typing
 * pauses, and the problems of the saved query and of a query not saved underlined with their message
 * on hover.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within, type Mock } from 'storybook/test'
import { onBeforeUnmount } from 'vue'
import type { TextRange } from '@numen/plugin-sdk'
import type { GetResultsResponse, UpdateSearchRequest, UpdateSearchResponse } from '../api/types'
import {
  COMPUTED,
  FAILED_LAST_RUN,
  FAILED_ON_RERANKER,
  FAILURE_REASON,
  FUNCTIONS,
  FUSED,
  HARBOR,
  HARBOR_TWO_LOGBOOK_PASSAGES,
  MISSPELLED,
  NEVER_COMPUTED,
  NOTE_PATH,
  NOTHING_FOUND,
  RERANKER_BUSY,
  RUNNING,
  STALE,
  WARNING,
} from '../fixtures/responses'
import { createScriptedService } from '../fixtures/service'
import { QUERY_PAUSE_MS } from '../model/queryEditor'
import { createSavedSearchTab } from '../model/tab'
import SavedSearchTab from './SavedSearchTab.vue'
import '../../styles.css'

interface Knobs {
  response: GetResultsResponse
  updateAnswer: UpdateSearchResponse
  onOpenAt: (path: string, ranges: readonly TextRange[]) => void
  onNotify: (text: string) => void
  onUpdateSearch: (request: UpdateSearchRequest) => void
  onGetResults: Mock<() => void>
}

const NOW = Date.parse('2026-11-14T09:45:00Z')

const NEW_QUERY = 'words("lighthouse") && meaning("the keeper in a gale").top(20)'

const meta: Meta<Knobs> = {
  title: 'Saved search/Tab',
  parameters: { layout: 'fullscreen' },
  args: {
    response: HARBOR,
    updateAnswer: { errors: [] },
    onOpenAt: fn(),
    onNotify: fn(),
    onUpdateSearch: fn(),
    onGetResults: fn(),
  },
  render: (args) => ({
    components: { SavedSearchTab },
    setup: () => {
      const service = createScriptedService(args.response, () => FUNCTIONS)
      service.setUpdateAnswer(args.updateAnswer)
      const updateSearch = service.updateSearch.bind(service)
      const getResults = service.getResults.bind(service)
      const tab = createSavedSearchTab(NOTE_PATH, {
        service: {
          ...service,
          getResults: (request, signal) => {
            args.onGetResults()
            return getResults(request, signal)
          },
          updateSearch: (request, signal) => {
            args.onUpdateSearch(request)
            return updateSearch(request, signal)
          },
        },
        openAt: (path, ranges) => args.onOpenAt(path, ranges),
        notify: (text) => args.onNotify(text),
        locale: 'en-GB',
        getNow: () => NOW,
      })
      tab.follow()
      service.watch.emit(COMPUTED)
      onBeforeUnmount(() => tab.dispose())
      return { state: tab.state }
    },
    template: `
      <div class="numen" style="block-size:40rem;background:var(--numen-surface);color:var(--numen-ink)">
        <SavedSearchTab :state="state" />
      </div>
    `,
  }),
}

export default meta
type Story = StoryObj<Knobs>

const FULL_TIME = /^14 Nov 2026, \d\d:\d\d$/

const getDot = (canvasElement: HTMLElement, name: string): Promise<HTMLElement> =>
  waitFor(() => within(canvasElement).getByRole('img', { name }))

/** The dot of fresh results, the edit button, two passages with their places and a whole file as a file row. */
export const ShowsTheResults: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await waitFor(() => expect(canvas.getByRole('button', { name: /1911/ })).toBeVisible())
    await expect(canvas.getByRole('button', { name: 'Edit query' })).toBeVisible()
    await expect(canvas.queryByText('meaning("the harbor master in a winter storm")')).toBeNull()
    await expect(await getDot(canvasElement, 'Up to date')).toBeVisible()
    await expect(canvas.queryByText('Computed')).toBeNull()
    await expect(canvas.getByRole('button', { name: /Night of the gale/ })).toBeVisible()
    await expect(canvas.getByText('Page 14')).toBeVisible()
    const file = canvas.getByRole('button', { name: /Pier plan/ })
    await expect(file).toHaveTextContent('Harbor/Pier plan.png')
    await expect(file).toHaveTextContent('image')
    await expect(canvas.queryByText('0.913')).toBeNull()
  },
}

/**
 * Hovering the dot shows the state as the title of a tooltip, and the time and the count of the results
 * with the full date on the time; leaving it hides it.
 */
export const ShowsTheTimeOfTheResultsOnHover: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const dot = await getDot(canvasElement, 'Up to date')

    await userEvent.hover(dot)

    const tooltip = await waitFor(() => canvas.getByRole('tooltip'))
    await expect(tooltip).toBeVisible()
    await expect(tooltip).toHaveTextContent(/^Up to date/)
    await expect(within(tooltip).getByText('Computed').nextElementSibling).toHaveTextContent(
      '15 minutes ago',
    )
    await expect(within(tooltip).getByTitle(FULL_TIME)).toHaveTextContent('15 minutes ago')
    await expect(within(tooltip).getByText('Results').nextElementSibling).toHaveTextContent('3')
    await expect(dot).toHaveAccessibleDescription(/Computed/)

    await userEvent.unhover(dot)
    await waitFor(() => expect(canvas.queryByRole('tooltip')).toBeNull())
  },
}

/** Focus on the dot shows its tooltip, Escape hides it. */
export const ShowsTheTooltipOnFocus: Story = {
  args: { response: STALE },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const dot = await getDot(canvasElement, 'Stale')

    await userEvent.click(canvas.getByRole('heading'))
    await userEvent.tab()
    await expect(dot).toHaveFocus()

    const tooltip = await waitFor(() => canvas.getByRole('tooltip'))
    await expect(tooltip).toHaveTextContent(/^Stale/)
    await expect(within(tooltip).getByTitle(FULL_TIME)).toBeVisible()

    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(canvas.queryByRole('tooltip')).toBeNull())
  },
}

/** A run in progress names the dot for the update. */
export const ShowsARunInProgress: Story = {
  args: { response: RUNNING },
  play: async ({ canvasElement }) => {
    await expect(await getDot(canvasElement, 'Updating')).toBeVisible()
  },
}

/** A click opens a passage at its range; Enter opens another; Space opens a whole file at its start. */
export const OpensAResultAtItsPlace: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const first = await waitFor(() => canvas.getByRole('button', { name: /1911/ }))

    await userEvent.click(first)
    await expect(args.onOpenAt).toHaveBeenLastCalledWith('Logbooks/1911.pdf', [
      { start: 48210, end: 50620 },
    ])

    canvas.getByRole('button', { name: /Night of the gale/ }).focus()
    await userEvent.keyboard('{Enter}')
    await expect(args.onOpenAt).toHaveBeenLastCalledWith('Harbor/Night of the gale.md', [
      { start: 0, end: 712 },
    ])

    canvas.getByRole('button', { name: /Pier plan/ }).focus()
    await userEvent.keyboard(' ')
    await expect(args.onOpenAt).toHaveBeenLastCalledWith('Harbor/Pier plan.png', [
      { start: 0, end: 0 },
    ])
  },
}

/** A filter no result holds says so with its text; Clear filter empties the field and brings the results back. */
export const FiltersTheResults: Story = {
  args: { response: HARBOR_TWO_LOGBOOK_PASSAGES },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const filter = await waitFor(() => canvas.getByRole('searchbox', { name: 'Filter results' }))

    await userEvent.type(filter, 'quay')

    await waitFor(() => expect(canvas.getAllByRole('listitem')).toHaveLength(1))
    await expect(canvas.getByText('Ice on the quay; no boat went out for a week.')).toBeVisible()

    await userEvent.clear(filter)
    await userEvent.type(filter, 'nothing like this')
    await waitFor(() =>
      expect(canvas.getByText('No result matches “nothing like this”')).toBeVisible(),
    )
    await expect(canvas.queryByRole('list', { name: 'Results' })).toBeNull()

    await userEvent.click(canvas.getByRole('button', { name: 'Clear filter' }))

    await waitFor(() => expect(canvas.getAllByRole('listitem')).toHaveLength(4))
    await expect(filter).toHaveValue('')
    await expect(canvas.queryByRole('button', { name: 'Clear filter' })).toBeNull()
  },
}

/**
 * The sources button is inside the filter field and opens the sources list: a search line and a row
 * per file, each checked at first; choosing a checked file hides its passages, choosing it again shows
 * them, and Escape puts the list away.
 */
export const HidesTheResultsOfASource: Story = {
  args: { response: HARBOR_TWO_LOGBOOK_PASSAGES },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const page = within(canvasElement.ownerDocument.body)
    const toggle = await waitFor(() => canvas.getByRole('button', { name: 'Sources' }))
    const field = canvas.getByRole('searchbox', { name: 'Filter results' }).parentElement
    await expect(field).toContainElement(toggle)

    await userEvent.click(toggle)
    await expect(toggle).toHaveAttribute('aria-expanded', 'true')
    const sources = await waitFor(() => page.getByRole('listbox', { name: 'Sources' }))
    const rows = within(sources).getAllByRole('option')
    await expect(rows).toHaveLength(3)
    for (const row of rows) await expect(row).toHaveAttribute('aria-current', 'true')

    await userEvent.type(page.getByRole('combobox', { name: 'Search sources' }), '1911')
    await waitFor(() => expect(within(sources).getAllByRole('option')).toHaveLength(1))
    const logbook = within(sources).getByRole('option', { name: /Logbooks\/1911\.pdf/ })

    await userEvent.click(logbook)

    const results = canvas.getByRole('list', { name: 'Results' })
    await waitFor(() => expect(within(results).getAllByRole('listitem')).toHaveLength(2))
    await expect(within(results).queryByRole('button', { name: /1911/ })).toBeNull()
    await expect(within(sources).getByRole('option', { name: /1911/ })).not.toHaveAttribute(
      'aria-current',
    )

    await userEvent.click(within(sources).getByRole('option', { name: /1911/ }))
    await waitFor(() => expect(within(results).getAllByRole('listitem')).toHaveLength(4))
    await expect(within(sources).getByRole('option', { name: /1911/ })).toHaveAttribute(
      'aria-current',
      'true',
    )

    await userEvent.keyboard('{Escape}')
    await waitFor(() => expect(toggle).toHaveAttribute('aria-expanded', 'false'))
    await expect(page.queryByRole('listbox', { name: 'Sources' })).toBeNull()
  },
}

/** Hiding every source says all sources are hidden; Show all sources brings every result back. */
export const ShowsAllSourcesAgain: Story = {
  args: { response: HARBOR_TWO_LOGBOOK_PASSAGES },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const page = within(canvasElement.ownerDocument.body)
    const toggle = await waitFor(() => canvas.getByRole('button', { name: 'Sources' }))

    await userEvent.click(toggle)
    const sources = await waitFor(() => page.getByRole('listbox', { name: 'Sources' }))
    for (const name of [/1911/, /Night of the gale/, /Pier plan/])
      await userEvent.click(within(sources).getByRole('option', { name }))
    await userEvent.keyboard('{Escape}')

    await waitFor(() => expect(canvas.getByText('All sources are hidden')).toBeVisible())
    await expect(canvas.queryByRole('list', { name: 'Results' })).toBeNull()

    await userEvent.click(canvas.getByRole('button', { name: 'Show all sources' }))

    await waitFor(() => expect(canvas.getAllByRole('listitem')).toHaveLength(4))
    await expect(canvas.queryByText('All sources are hidden')).toBeNull()
  },
}

/**
 * A query the core rejects: the red dot says the query has problems, the problems themselves only in the
 * editor, and the results of the last good run.
 */
export const ShowsTheProblemsOfTheQueryWithTheLastGoodResults: Story = {
  args: { response: MISSPELLED },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const dot = await getDot(canvasElement, 'Failed')

    await expect(canvas.queryByRole('alert')).toBeNull()
    await expect(canvas.getByRole('button', { name: /1911/ })).toBeVisible()

    await userEvent.hover(dot)
    const tooltip = await waitFor(() => canvas.getByRole('tooltip'))
    await expect(tooltip).toHaveTextContent('the query has problems')
    await expect(tooltip).not.toHaveTextContent('meanin')
  },
}

/** A run that found nothing says so, with no button to bring results back. */
export const SaysWhenNothingMatches: Story = {
  args: { response: NOTHING_FOUND },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await waitFor(() => expect(canvas.getByText('The query found nothing')).toBeVisible())
    await expect(canvas.queryByRole('list')).toBeNull()
    await expect(canvas.queryByRole('button', { name: 'Clear filter' })).toBeNull()
    await expect(canvas.queryByRole('button', { name: 'Show all sources' })).toBeNull()
  },
}

/** A note never computed names its dot for the computing, and does not say nothing matches. */
export const SaysTheResultsAreBeingComputed: Story = {
  args: { response: NEVER_COMPUTED },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await expect(await getDot(canvasElement, 'Computing')).toBeVisible()
    await expect(canvas.queryByText('The query found nothing')).toBeNull()
  },
}

/** A warning of the core turns the dot yellow and shows in its tooltip, nowhere else. */
export const ShowsAWarning: Story = {
  args: { response: FUSED },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const dot = await getDot(canvasElement, 'With warnings')

    await expect(canvas.queryByText(WARNING)).toBeNull()
    await userEvent.hover(dot)
    const tooltip = await waitFor(() => canvas.getByRole('tooltip'))
    await expect(within(tooltip).getByRole('heading', { name: 'Warning' })).toBeVisible()
    await expect(tooltip).toHaveTextContent(WARNING)
    await expect(canvas.getByRole('button', { name: /1911/ })).toBeVisible()
  },
}

/**
 * A failed last run names the red dot for it; the tooltip gives the time of the good run, whose results
 * stay, and the time of the failed run, and its reason under the Error heading.
 */
export const ShowsAFailedLastRun: Story = {
  args: { response: FAILED_LAST_RUN },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const dot = await getDot(canvasElement, 'Failed')

    await expect(canvas.queryByText(new RegExp(FAILURE_REASON))).toBeNull()
    await userEvent.hover(dot)

    const tooltip = await waitFor(() => canvas.getByRole('tooltip'))
    await expect(tooltip).toHaveTextContent(/^Failed/)
    await expect(within(tooltip).getAllByTitle(FULL_TIME)).toHaveLength(2)
    await expect(
      within(tooltip).getAllByText('Failed').at(-1)?.nextElementSibling,
    ).toHaveTextContent('4 minutes ago')
    await expect(within(tooltip).getByRole('heading', { name: 'Error' })).toBeVisible()
    await expect(tooltip).toHaveTextContent(FAILURE_REASON)
    await expect(canvas.getByRole('button', { name: /1911/ })).toBeVisible()
    await expect(canvas.getByRole('button', { name: /Pier plan/ })).toBeVisible()
  },
}

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
