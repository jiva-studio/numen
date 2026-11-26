/**
 * The tab of a saved search note over a scripted service: the query and its results, a passage and a
 * whole file opened, the dot of the state of the run and its tooltip, the problems of the query with the
 * last good results, no result, a warning, a failed last run, and the filter and the sources of the
 * results and the button that brings back what they hide.
 */
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { expect, userEvent, waitFor, within } from 'storybook/test'
import {
  FAILED_LAST_RUN,
  FAILURE_REASON,
  FUSED,
  HARBOR_TWO_LOGBOOK_PASSAGES,
  MISSPELLED,
  NEVER_COMPUTED,
  NOTHING_FOUND,
  RUNNING,
  STALE,
  WARNING,
} from '../fixtures/responses'
import { getDot, renderTab, TAB_ARGS, type Knobs } from '../fixtures/tabStory'
import type { GetResultsResponse } from '../api/types'

const meta: Meta<Knobs> = {
  title: 'Saved search/Tab',
  parameters: { layout: 'fullscreen' },
  args: TAB_ARGS,
  render: renderTab,
}

export default meta
type Story = StoryObj<Knobs>

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
 * Hovering the dot shows the state as the title of a tooltip, and the time and the count of the results;
 * leaving it hides it.
 */
export const ShowsTheTimeOfTheResultsOnHover: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement)
    const dot = await getDot(canvasElement, 'Up to date')

    await userEvent.hover(dot)

    const tooltip = await waitFor(() => canvas.getByRole('tooltip'))
    await waitFor(() => expect(tooltip).toBeVisible())
    await expect(tooltip).toHaveTextContent(/^Up to date/)
    await expect(within(tooltip).getByText('Computed').nextElementSibling).toHaveTextContent(
      '15 minutes ago',
    )
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
    await waitFor(() => expect(within(tooltip).getByText('15 minutes ago')).toBeVisible())

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
    await waitFor(() =>
      expect(within(tooltip).getByRole('heading', { name: 'Warning' })).toBeVisible(),
    )
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
    await expect(within(tooltip).getByText('Computed').nextElementSibling).toHaveTextContent(
      '15 minutes ago',
    )
    await expect(
      within(tooltip).getAllByText('Failed').at(-1)?.nextElementSibling,
    ).toHaveTextContent('4 minutes ago')
    await waitFor(() =>
      expect(within(tooltip).getByRole('heading', { name: 'Error' })).toBeVisible(),
    )
    await expect(tooltip).toHaveTextContent(FAILURE_REASON)
    await expect(canvas.getByRole('button', { name: /1911/ })).toBeVisible()
    await expect(canvas.getByRole('button', { name: /Pier plan/ })).toBeVisible()
  },
}

const result = (path: string, kind: string, text: string, score: number, page?: number) => ({
  path,
  title: path
    .split('/')
    .pop()!
    .replace(/\.\w+$/, ''),
  kind,
  ...(page ? { place: { page } } : {}),
  text,
  start: 0,
  length: text.length,
  line: 1,
  uri: `numen:${encodeURI(path)}`,
  score,
})

/** A saved search of a vault of physics notes: what erasing a memory costs. */
const ERASING: GetResultsResponse = {
  ...FUSED,
  path: 'Physics/What erasing costs.md',
  title: 'What erasing costs',
  query: 'kind in ["note", "pdf"]\n&& meaning("what it costs to erase a bit of memory")\n.top(30)',
  order: 'reranked',
  warnings: [],
  results: [
    result(
      "Computation/Landauer's principle.md",
      'note',
      'Clearing one bit of memory costs at least kT ln 2 of heat, whatever the memory is made of.',
      0.93,
    ),
    result(
      'Reading/Boltzmann 1877.pdf',
      'pdf',
      '…the clearing is the payment. It is exacted not at the door but at the moment the record of a molecule is destroyed…',
      0.88,
      374,
    ),
    result(
      "Physics/Maxwell's demon.md",
      'note',
      'The memory fills, and clearing it costs what Landauer says a bit costs.',
      0.84,
    ),
    result(
      'Reading/Shannon 1948.pdf',
      'pdf',
      '…the entropy of the source is the logarithm of the number of messages it could have sent…',
      0.71,
      11,
    ),
  ],
}

/** The tab of a saved search in the window, its results under the query. */
export const InTheWindow: Story = {
  tags: ['screenshot'],
  parameters: {
    frame: {
      place: 'tab',
      tab: { title: 'What erasing costs', icon: 'search' },
      note: 'after',
      share: 0.58,
    },
    screenshot: [{ name: 'results' }],
  },
  args: { response: ERASING },
  play: async ({ canvasElement }) => {
    await waitFor(() =>
      expect(
        within(canvasElement).getByRole('button', { name: /^Landauer's principle/ }),
      ).toBeVisible(),
    )
  },
}
