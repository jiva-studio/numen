/**
 * The tab of a saved search note over a scripted service: the query and its results, a passage and a
 * whole file opened, the problems of the query with the last good results, no result, a warning, a
 * failed last run, the filter and the sources of the results, and the query edited in the code editor
 * the edit button opens: completion from the catalog, the problems underlined, Save and Cancel.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, fn, userEvent, waitFor, within } from 'storybook/test'
import { onBeforeUnmount } from 'vue'
import type { TextRange } from '@numen/plugin-sdk'
import type { GetResultsResponse, UpdateSearchRequest, UpdateSearchResponse } from '../api/types'
import {
  COMPUTED,
  FAILED_LAST_RUN,
  FAILURE_REASON,
  FUNCTIONS,
  FUSED,
  HARBOR,
  HARBOR_TWO_LOGBOOK_PASSAGES,
  MISSPELLED,
  NEVER_COMPUTED,
  NOTE_PATH,
  NOTHING_FOUND,
  WARNING,
} from '../fixtures/responses'
import { createScriptedService } from '../fixtures/service'
import { createSavedSearchTab } from '../model/tab'
import SavedSearchTab from './SavedSearchTab.vue'
import '../../styles.css'

interface Knobs {
  response: GetResultsResponse
  updateAnswer: UpdateSearchResponse
  onOpenAt: (path: string, ranges: readonly TextRange[]) => void
  onNotify: (text: string) => void
  onUpdateSearch: (request: UpdateSearchRequest) => void
}

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
  },
  render: (args) => ({
    components: { SavedSearchTab },
    setup: () => {
      const service = createScriptedService(args.response)
      service.setUpdateAnswer(args.updateAnswer)
      service.setFunctions(FUNCTIONS)
      const updateSearch = service.updateSearch.bind(service)
      const tab = createSavedSearchTab(NOTE_PATH, {
        service: {
          ...service,
          updateSearch: (request, signal) => {
            args.onUpdateSearch(request)
            return updateSearch(request, signal)
          },
        },
        openAt: (path, ranges) => args.onOpenAt(path, ranges),
        notify: (text) => args.onNotify(text),
        locale: 'en-GB',
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

/** The status, the edit button, two passages with their places and a whole file as a file row. */
export const ShowsTheResults: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await waitFor(() => expect(canvas.getByRole('button', { name: /1911/ })).toBeVisible())
    await expect(canvas.getByRole('button', { name: 'Edit query' })).toBeVisible()
    await expect(canvas.queryByText('meaning("the harbor master in a winter storm")')).toBeNull()
    await expect(canvas.getByText(/^Computed 14 Nov 2026, \d\d:\d\d$/)).toBeVisible()
    await expect(canvas.getByRole('button', { name: /Night of the gale/ })).toBeVisible()
    await expect(canvas.getByText('Page 14')).toBeVisible()
    const file = canvas.getByRole('button', { name: /Pier plan/ })
    await expect(file).toHaveTextContent('Harbor/Pier plan.png')
    await expect(file).toHaveTextContent('image')
    await expect(canvas.queryByText('0.913')).toBeNull()
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

/** Typing in the filter narrows the results to those whose text, title or file name holds it. */
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
    await waitFor(() => expect(canvas.getByText('No result matches the filter.')).toBeVisible())
  },
}

/**
 * The sources list has a search line and a row per file, each checked at first; choosing a checked
 * file hides its passages, choosing it again shows them, and Escape puts the list away.
 */
export const HidesTheResultsOfASource: Story = {
  args: { response: HARBOR_TWO_LOGBOOK_PASSAGES },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const page = within(canvasElement.ownerDocument.body)
    const toggle = await waitFor(() => canvas.getByRole('button', { name: 'Sources' }))

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

/** A query the core rejects: its problems as alerts, and the results of the last good run. */
export const ShowsTheProblemsOfTheQueryWithTheLastGoodResults: Story = {
  args: { response: MISSPELLED },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await waitFor(() =>
      expect(canvas.getByRole('alert')).toHaveTextContent(
        '9:4 unknown function meanin (did you mean meaning?)',
      ),
    )
    await expect(canvas.getByRole('button', { name: /1911/ })).toBeVisible()
  },
}

/** A run that found nothing says so. */
export const SaysWhenNothingMatches: Story = {
  args: { response: NOTHING_FOUND },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await waitFor(() => expect(canvas.getByText('No passage matches the query.')).toBeVisible())
    await expect(canvas.queryByRole('list')).toBeNull()
  },
}

/** A note never computed says its results are being computed, and does not say nothing matches. */
export const SaysTheResultsAreBeingComputed: Story = {
  args: { response: NEVER_COMPUTED },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await waitFor(() => expect(canvas.getByText('Computing the results…')).toBeVisible())
    await expect(canvas.queryByText('No passage matches the query.')).toBeNull()
  },
}

/** A warning of the core shows beside the results. */
export const ShowsAWarning: Story = {
  args: { response: FUSED },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await waitFor(() =>
      expect(
        canvas.getAllByRole('status').some((one) => one.textContent?.trim() === WARNING),
      ).toBe(true),
    )
    await expect(canvas.getByRole('button', { name: /1911/ })).toBeVisible()
  },
}

/** A failed last run is named with its time and reason; the results of the good run stay. */
export const ShowsAFailedLastRun: Story = {
  args: { response: FAILED_LAST_RUN },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await waitFor(() =>
      expect(
        canvas.getByText(
          new RegExp(`^Last run failed 14 Nov 2026, \\d\\d:\\d\\d: ${FAILURE_REASON}$`),
        ),
      ).toBeVisible(),
    )
    await expect(canvas.getByText(/^Computed 14 Nov 2026, \d\d:\d\d$/)).toBeVisible()
    await expect(canvas.getByRole('button', { name: /1911/ })).toBeVisible()
    await expect(canvas.getByRole('button', { name: /Pier plan/ })).toBeVisible()
  },
}

const getEditor = (canvasElement: HTMLElement): HTMLElement =>
  within(canvasElement).getByRole('textbox', { name: 'Query' })

const openEditor = async (canvasElement: HTMLElement): Promise<HTMLElement> => {
  const canvas = within(canvasElement)
  await userEvent.click(await waitFor(() => canvas.getByRole('button', { name: 'Edit query' })))
  await waitFor(() => expect(canvas.getByRole('dialog', { name: 'Edit query' })).toBeVisible())
  return getEditor(canvasElement)
}

/** The browser types the text into the code editor: it takes text only from real keystrokes. */
const fillEditor = async (editor: HTMLElement, text: string): Promise<void> => {
  const browser = await import('vitest/browser')
  await browser.userEvent.fill(editor, text)
}

/** The edit button opens the query in the editor; Save writes it, closes the editor and keeps the new query. */
export const EditsAndSavesTheQuery: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const editor = await openEditor(canvasElement)
    await expect(editor).toHaveTextContent('meaning("the harbor master in a winter storm")')

    await fillEditor(editor, NEW_QUERY)
    await userEvent.click(canvas.getByRole('button', { name: 'Save' }))

    await expect(args.onUpdateSearch).toHaveBeenCalledWith({ path: NOTE_PATH, query: NEW_QUERY })
    await waitFor(() => expect(canvas.queryByRole('dialog')).toBeNull())
    await expect(await openEditor(canvasElement)).toHaveTextContent(NEW_QUERY)
  },
}

/** Cancel closes the editor and writes nothing. */
export const CancelsTheEdit: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await fillEditor(await openEditor(canvasElement), NEW_QUERY)

    await userEvent.click(canvas.getByRole('button', { name: 'Cancel' }))

    await waitFor(() => expect(canvas.queryByRole('dialog')).toBeNull())
    await expect(args.onUpdateSearch).not.toHaveBeenCalled()
    await expect(await openEditor(canvasElement)).toHaveTextContent(
      'meaning("the harbor master in a winter storm")',
    )
  },
}

/** Typing a name offers the functions of the catalog with their description; after a dot, the stages. */
export const CompletesTheNamesOfTheCatalog: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const browser = await import('vitest/browser')
    const editor = await openEditor(canvasElement)

    await fillEditor(editor, 'words("gale") && t')
    await browser.userEvent.keyboard('a')

    const list = await waitFor(() => canvas.getByRole('listbox'))
    await waitFor(() =>
      expect(within(list).getAllByRole('option').map((one) => one.textContent)).toStrictEqual([
        expect.stringContaining('tags'),
      ]),
    )
    await waitFor(() => expect(canvas.getByText(/Notes that carry the tag\./)).toBeVisible())
    await expect(canvas.getByText(/tags\("harbor"\)/)).toBeVisible()

    await browser.userEvent.keyboard('{Enter}')
    await expect(editor).toHaveTextContent('words("gale") && tags')

    await browser.userEvent.keyboard('("harbor").r')
    const stages = await waitFor(() => canvas.getByRole('listbox'))
    await waitFor(() =>
      expect(within(stages).getAllByRole('option').map((one) => one.textContent)).toStrictEqual([
        expect.stringContaining('rerank'),
      ]),
    )
  },
}

/** A query the core rejects keeps the editor open, lists its problems and underlines each at its place. */
export const UnderlinesTheProblemsOfAQueryNotSaved: Story = {
  args: {
    updateAnswer: {
      errors: [
        { line: 1, column: 18, message: 'unknown function meanin (did you mean meaning?)' },
      ],
    },
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const editor = await openEditor(canvasElement)

    await fillEditor(editor, 'words("gale") && meanin("a night at sea")')
    await userEvent.click(canvas.getByRole('button', { name: 'Save' }))

    await expect(args.onUpdateSearch).toHaveBeenCalledTimes(1)
    await waitFor(() =>
      expect(canvas.getByRole('alert')).toHaveTextContent(
        '1:18 unknown function meanin (did you mean meaning?)',
      ),
    )
    const underlined = await waitFor(() => {
      const marks = canvasElement.querySelectorAll('.cm-lintRange-error')
      expect(marks).toHaveLength(1)
      return marks[0]!
    })
    await expect(underlined).toHaveTextContent(/^meanin$/)
    await expect(canvas.getByRole('dialog', { name: 'Edit query' })).toBeVisible()
  },
}
