/** The panes of the workspace and the tabs they hold. */
import { expect, type Locator, type Page } from 'playwright/test'

const PANES = 'section[data-workspace-pane]'

export class Panes {
  constructor(private readonly page: Page) {}

  /** The two panes in the order of the window; a window with fewer than two has no other pane to move to. */
  async listTwo(): Promise<Locator[]> {
    const panes = this.page.locator(PANES)
    await expect(panes).toHaveCount(2)
    return panes.all()
  }

  getTabIn(pane: Locator, tabName: RegExp): Locator {
    return pane.getByRole('tab', { name: tabName })
  }

  /** The panes that hold a tab with the name. */
  getHolding(tabName: RegExp): Locator {
    return this.page.locator(PANES).filter({ has: this.page.getByRole('tab', { name: tabName }) })
  }

  getTabs(pane: Locator): Locator {
    return pane.getByRole('tab')
  }
}
