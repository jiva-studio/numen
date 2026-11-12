/** The title bar of the window: its buttons and the notice of an available update. */
import type { Locator, Page } from 'playwright/test'

const UPDATE_BUTTON = /^Update available: v/
const UPDATE_NOTICE = 'Update available'

export class TitleBar {
  constructor(private readonly page: Page) {}

  getButton(label: string): Locator {
    return this.page.locator('[data-title-bar]').getByRole('button', { name: label, exact: true })
  }

  getUpdateButton(): Locator {
    return this.page.getByRole('button', { name: UPDATE_BUTTON })
  }

  getUpdateNotice(): Locator {
    return this.page.getByRole('dialog', { name: UPDATE_NOTICE })
  }

  getNoticeButton(label: string): Locator {
    return this.getUpdateNotice().getByRole('button', { name: label, exact: true })
  }
}
