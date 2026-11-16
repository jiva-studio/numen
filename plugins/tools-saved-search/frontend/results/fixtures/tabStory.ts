/**
 * What the stories of the saved search tab share: the scripted service behind the tab, the knobs of
 * a story and the dot of the state of the run.
 */
import type { Meta } from '@storybook/vue3-vite'
import { fn, waitFor, within, type Mock } from 'storybook/test'
import { onBeforeUnmount } from 'vue'
import type { TextRange } from '@numen/plugin-sdk'
import type { GetResultsResponse, UpdateSearchRequest, UpdateSearchResponse } from '../api/types'
import { COMPUTED, FUNCTIONS, HARBOR, NOTE_PATH } from './responses'
import { createScriptedService } from './service'
import { createSavedSearchTab } from '../model/tab'
import SavedSearchTab from '../ui/SavedSearchTab.vue'
import '../../styles.css'

export interface Knobs {
  response: GetResultsResponse
  updateAnswer: UpdateSearchResponse
  onOpenAt: (path: string, ranges: readonly TextRange[]) => void
  onNotify: (text: string) => void
  onUpdateSearch: (request: UpdateSearchRequest) => void
  onGetResults: Mock<() => void>
}

export const NOW = Date.parse('2026-11-14T09:45:00Z')

export const createTabMeta = (title: string): Meta<Knobs> => ({
  title,
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
})

export const getDot = (canvasElement: HTMLElement, name: string): Promise<HTMLElement> =>
  waitFor(() => within(canvasElement).getByRole('img', { name }))
