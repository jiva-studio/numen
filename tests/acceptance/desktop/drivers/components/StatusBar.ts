/** The status bar at the bottom of the window. */
import type { Locator, Page } from 'playwright/test'

export class StatusBar {
  constructor(private readonly page: Page) {}

  get root(): Locator {
    return this.page.getByRole('region', { name: 'Status bar' })
  }
}
