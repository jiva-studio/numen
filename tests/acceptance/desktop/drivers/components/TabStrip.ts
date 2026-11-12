/** The tabs of the window: finding one by id, title, file or kind, and bringing it to the front. */
import { expect, type Locator, type Page } from 'playwright/test'
import { escapeRegExp, getStem, getTitlePattern } from '../rules.js'
import type { Host } from './host.js'

export class TabStrip {
  constructor(private readonly host: Host) {}

  /** The tab with the id the core gave it. */
  tabById(id: string): Locator {
    return this.host.page.getByRole('tab').and(this.host.page.locator(`[data-workspace-tab="${id}"]`))
  }

  /** The tab of a note: a tab linked to the file tree is named with the words "Linked tab" before its title. */
  noteTab(title: string): Locator {
    const escaped = escapeRegExp(title)
    return this.host.page
      .getByRole('tab', { name: new RegExp(`^(Linked tab )?${escaped} Close ${escaped}$`) })
      .and(this.host.page.locator('[data-workspace-tab^="note:"]'))
  }

  noteTabs(): Locator {
    return this.host.page.locator('[data-workspace-tab^="note:"]')
  }

  /** The tabs titled by the words. */
  tabTitled(title: string): Locator {
    return this.host.page.getByRole('tab', { name: getTitlePattern(title) })
  }

  /** The tab of the file, named by the beginning of its file name. */
  tabOfFile(file: string): Locator {
    return this.tabOfFileIn(this.host.page, file)
  }

  /** The tab of the file inside a pane. */
  tabOfFileIn(pane: Locator | Page, file: string): Locator {
    return pane.getByRole('tab', { name: new RegExp('^' + escapeRegExp(getStem(file)), 'i') })
  }

  /** The first pane of the window. */
  firstPane(): Locator {
    return this.host.page.locator('section[data-workspace-pane]').first()
  }

  closeButton(tab: Locator): Locator {
    return tab.getByRole('button', { name: /^Close / })
  }

  /** Whether the tab is the one in front of its pane. */
  async isSelected(tab: Locator): Promise<boolean> {
    return (await tab.getAttribute('aria-selected')) === 'true'
  }

  /** Brings the tab to the front of its pane. */
  async select(tab: Locator): Promise<void> {
    if (!(await this.isSelected(tab))) await tab.click()
  }

  /** Waits for the tab to show, then brings it to the front. */
  async show(tab: Locator): Promise<void> {
    await expect(tab).toBeVisible()
    await this.select(tab)
  }
}
