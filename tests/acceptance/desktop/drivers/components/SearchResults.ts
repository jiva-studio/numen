/** The results of the search of the vault, grouped by kind, and the list of actions of one result. */
import { type Locator } from 'playwright/test'
import { getEscaped } from '../rules.js'
import type { Host } from './host.js'
import { Palette } from './Palette.js'

export class SearchResults {
  private readonly palette: Palette

  constructor(private readonly host: Host) {
    this.palette = new Palette(host)
  }

  /** The results of the group that have the text. */
  results(text: string, group: string): Locator {
    return this.host.page.getByRole('group', { name: group }).getByRole('option').filter({ hasText: text })
  }

  /** The result named exactly by the words, or the first result that has them when no name is the same. */
  async pickRow(results: Locator, text: string): Promise<Locator> {
    const named = results.filter({ has: this.palette.nameCell(new RegExp(`^${getEscaped(text)}$`)) })
    return (await named.count()) > 0 ? named.first() : results.first()
  }

  /** Every result of the open list. */
  rows(): Locator {
    return this.palette.rows()
  }

  /** The groups that have not answered yet. */
  busyGroups(): Locator {
    return this.palette.busyGroups()
  }

  /** The rows of the list of actions of the lit result. */
  actionRows(): Locator {
    return this.host.page.locator('[data-actions="list"] [role="option"]')
  }

  /** The names of the actions, as they are shown. */
  async readActionNames(): Promise<string[]> {
    const names = await this.host.page.locator('[data-actions="list"] [data-actions="name"]').allTextContents()
    return names.map((one) => one.trim())
  }

  /** The action rows named exactly by the words. */
  actionNamed(action: string): Locator {
    const name = new RegExp('^' + getEscaped(action) + '$')
    return this.actionRows().filter({ has: this.host.page.locator('[data-actions="name"]', { hasText: name }) })
  }
}
