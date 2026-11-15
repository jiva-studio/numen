/**
 * The tab of a saved search note over a scripted service: the query and its results, a passage and a
 * whole file opened, the problems of the query with the last good results, no result, a warning, a
 * failed last run, the filter and the sources of the results, and the query edited and saved in the
 * accordion at the top.
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

/** The query, the status, two passages with their places and a whole file as a file row. */
export const ShowsTheQueryAndTheResults: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await waitFor(() => expect(canvas.getByRole('button', { name: /1911/ })).toBeVisible())
    await expect(
      canvas.getByText('meaning("the harbor master in a winter storm")', { exact: false }),
    ).toBeVisible()
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

/** The accordion opens the query alone; Save writes it and shows the new query. */
export const EditsAndSavesTheQuery: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const opener = await waitFor(() => canvas.getByRole('button', { name: 'Edit query' }))
    await expect(opener).toHaveAttribute('aria-expanded', 'false')

    await userEvent.click(opener)
    await expect(opener).toHaveAttribute('aria-expanded', 'true')
    const query = canvas.getByRole('textbox', { name: 'Query' })
    await expect(query).toHaveValue(HARBOR.query)
    await expect(canvas.queryByRole('textbox', { name: 'Rerank' })).toBeNull()
    await expect(canvas.queryByRole('spinbutton', { name: 'Limit' })).toBeNull()

    await userEvent.clear(query)
    await userEvent.type(query, NEW_QUERY)
    await userEvent.click(canvas.getByRole('button', { name: 'Save' }))

    await expect(args.onUpdateSearch).toHaveBeenCalledWith({ path: NOTE_PATH, query: NEW_QUERY })
    await waitFor(() => expect(canvas.getByText(NEW_QUERY, { exact: false })).toBeVisible())
  },
}

/** A query the core rejects shows its problems in the accordion and leaves the note as it was. */
export const ShowsTheProblemsOfAQueryNotSaved: Story = {
  args: {
    updateAnswer: {
      errors: [{ line: 1, column: 1, message: 'unknown function meanin (did you mean meaning?)' }],
    },
  },
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    await userEvent.click(await waitFor(() => canvas.getByRole('button', { name: 'Edit query' })))
    const query = canvas.getByRole('textbox', { name: 'Query' })

    await userEvent.clear(query)
    await userEvent.type(query, 'meanin("gale")')
    await userEvent.click(canvas.getByRole('button', { name: 'Save' }))

    await expect(args.onUpdateSearch).toHaveBeenCalledTimes(1)
    await waitFor(() =>
      expect(canvas.getByRole('alert')).toHaveTextContent(
        '1:1 unknown function meanin (did you mean meaning?)',
      ),
    )
    await expect(
      canvas.getByText('meaning("the harbor master in a winter storm")', { exact: false }),
    ).toBeVisible()
  },
}
