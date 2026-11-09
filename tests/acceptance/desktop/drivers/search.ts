/** The person at the search of the vault: typing words, and the results grouped by kind. */
import { expect, type Locator } from 'playwright/test'
import { getPaletteField } from '../../../e2e/desktop/tests/support/workspace.js'
import type { Scenario } from './scenario.js'

/** The first pass over a new file may take this long before the search finds it. */
const INDEXING_LIMIT_MS = 90_000

export class SearchDriver {
  private query = ''

  constructor(private readonly scenario: Scenario) {}

  private async type(): Promise<void> {
    const { page } = this.scenario
    const field = getPaletteField(page)
    if (!(await field.isVisible())) await page.keyboard.press('Control+k')
    await field.fill('')
    await field.fill(this.query)
  }

  async search(query: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    this.query = query
    await this.type()
  }

  private getResults(text: string, group: string): Locator {
    return this.scenario.page.getByRole('group', { name: group }).getByRole('option').filter({ hasText: text })
  }

  private async waitForResult(text: string, group: string): Promise<Locator> {
    const results = this.getResults(text, group)
    await expect(async () => {
      await this.type()
      await expect(results.first()).toBeVisible({ timeout: 3_000 })
    }).toPass({ timeout: INDEXING_LIMIT_MS })
    return results.first()
  }

  async expectResult(text: string, group: string): Promise<void> {
    await this.waitForResult(text, group)
  }

  async expectNoResult(text: string, group: string): Promise<void> {
    await expect(this.getResults(text, group)).toHaveCount(0)
  }

  async chooseResult(text: string, group: string): Promise<void> {
    await (await this.waitForResult(text, group)).click()
  }
}
