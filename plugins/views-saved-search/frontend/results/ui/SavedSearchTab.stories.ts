/**
 * The tab of a saved search note over a scripted service: the query and its results, a result
 * opened at its place, the problems of the query, no result, a warning, a failed last run, and the
 * query edited and saved in the accordion at the top.
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
  FUSED_WARNING,
  HARBOR,
  MISSPELLED,
  NEVER_COMPUTED,
  NOTE_PATH,
  NOTHING_FOUND,
} from '../fixtures/responses'
import { createScriptedService } from '../fixtures/service'
import { createSavedSearchTab } from '../model/tab'
import SavedSearchTab from './SavedSearchTab.vue'
import '../../styles.css'

interface Knobs {
  response: GetResultsResponse
  updateAnswer: UpdateSearchResponse
  onOpenAt: (path: string, ranges: readonly TextRange[]) => void
  onUpdateSearch: (request: UpdateSearchRequest) => void
}

const NEW_QUERY = 'words("lighthouse") && meaning("the keeper in a gale")'
const NEW_RERANK = 'The keeper of the lighthouse on a night of gale.'

const meta: Meta<Knobs> = {
  title: 'Saved search/Tab',
  parameters: { layout: 'fullscreen' },
  args: {
    response: HARBOR,
    updateAnswer: { errors: [] },
    onOpenAt: fn(),
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
        locale: 'en-GB',
      })
      tab.follow()
      service.emit(COMPUTED)
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

/** The query, the rerank text, the status and three results with their places. */
export const ShowsTheQueryAndTheResults: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await waitFor(() => expect(canvas.getByRole('button', { name: /1911/ })).toBeVisible())
    await expect(
      canvas.getByText('meaning("the harbor master in a winter storm")', { exact: false }),
    ).toBeVisible()
    await expect(canvas.getByText('Rerank:', { exact: false })).toBeVisible()
    await expect(canvas.getByText(HARBOR.rerank, { exact: false })).toBeVisible()
    await expect(canvas.getByText(/^Computed 14 Nov 2026, \d\d:\d\d$/)).toBeVisible()
    await expect(canvas.getByRole('button', { name: /Night of the gale/ })).toBeVisible()
    await expect(canvas.getByRole('button', { name: /Pier talk/ })).toBeVisible()
    await expect(canvas.getByText('Page 14 of 212')).toBeVisible()
    await expect(canvas.getByText('12:34–13:32')).toBeVisible()
    await expect(canvas.queryByText('0.913')).toBeNull()
  },
}

/** A click opens the first result at its range; Enter and Space open the others. */
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

    canvas.getByRole('button', { name: /Pier talk/ }).focus()
    await userEvent.keyboard(' ')
    await expect(args.onOpenAt).toHaveBeenLastCalledWith('Recordings/Pier talk.m4a', [
      { start: 10233, end: 11413 },
    ])
  },
}

/** A query that does not compile: its problems as alerts, and no result list. */
export const ShowsTheProblemsOfTheQuery: Story = {
  args: { response: MISSPELLED },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await waitFor(() =>
      expect(canvas.getByRole('alert')).toHaveTextContent("9:4 undeclared reference to 'meanin'"),
    )
    await expect(canvas.queryByRole('list')).toBeNull()
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

/** A run kept in fused order shows its warning beside the results. */
export const ShowsAWarning: Story = {
  args: { response: FUSED },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)

    await waitFor(() =>
      expect(
        canvas.getAllByRole('status').some((one) => one.textContent?.trim() === FUSED_WARNING),
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
    await expect(canvas.getByRole('button', { name: /Pier talk/ })).toBeVisible()
  },
}

/** The accordion opens the query, the rerank text and the limit; Save writes them and shows the new query. */
export const EditsAndSavesTheQuery: Story = {
  play: async ({ args, canvasElement }) => {
    const canvas = within(canvasElement)
    const opener = await waitFor(() => canvas.getByRole('button', { name: 'Edit query' }))
    await expect(opener).toHaveAttribute('aria-expanded', 'false')

    await userEvent.click(opener)
    await expect(opener).toHaveAttribute('aria-expanded', 'true')
    const query = canvas.getByRole('textbox', { name: 'Query' })
    const rerank = canvas.getByRole('textbox', { name: 'Rerank' })
    const limit = canvas.getByRole('spinbutton', { name: 'Limit' })
    await expect(query).toHaveValue(HARBOR.query)
    await expect(rerank).toHaveValue(HARBOR.rerank)
    await expect(limit).toHaveValue(30)

    await userEvent.clear(query)
    await userEvent.type(query, NEW_QUERY)
    await userEvent.clear(rerank)
    await userEvent.type(rerank, NEW_RERANK)
    await userEvent.clear(limit)
    await userEvent.type(limit, '20')
    await userEvent.click(canvas.getByRole('button', { name: 'Save' }))

    await expect(args.onUpdateSearch).toHaveBeenCalledWith({
      path: NOTE_PATH,
      query: NEW_QUERY,
      rerank: NEW_RERANK,
      limit: 20,
    })
    await waitFor(() => expect(canvas.getByText(NEW_QUERY, { exact: false })).toBeVisible())
  },
}

/** A query the backend refuses shows its problems in the accordion and leaves the note as it was. */
export const ShowsTheProblemsOfAQueryNotSaved: Story = {
  args: {
    updateAnswer: {
      errors: [{ line: 1, column: 1, message: "undeclared reference to 'meanin'" }],
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
      expect(canvas.getByRole('alert')).toHaveTextContent("1:1 undeclared reference to 'meanin'"),
    )
    await expect(
      canvas.getByText('meaning("the harbor master in a winter storm")', { exact: false }),
    ).toBeVisible()
  },
}
