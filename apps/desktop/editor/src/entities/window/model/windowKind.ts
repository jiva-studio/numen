import { hasWindow, win } from '@numen/ui'

const getSearch = (): string => (hasWindow() ? win.location.search : '')

/** Whether this window was made to show a tab detached or moved from another window. */
export function isSecondaryWindow(search: string = getSearch()): boolean {
  const params = new URLSearchParams(search)
  return params.get('window') === 'secondary' || params.has('secondary')
}

/** The id of the tab a window made for a detach shows, empty where this is not such a window. */
export function getDetachedTabId(search: string = getSearch()): string {
  return new URLSearchParams(search).get('id') ?? ''
}
