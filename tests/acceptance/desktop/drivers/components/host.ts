import type { Page } from 'playwright/test'

/** What a component needs from its owner: the page, read at each use, as a scenario may restart its window. */
export interface Host {
  readonly page: Page
}
