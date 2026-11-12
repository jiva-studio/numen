/** The person at the search of the vault: typing words, and the results grouped by kind. */
import { expect, type Locator } from 'playwright/test'
import { Palette } from './components/Palette.js'
import { SearchResults } from './components/SearchResults.js'
import { parseColumn } from './rules.js'
import type { Scenario } from './scenario.js'

/** The first pass over a new file may take this long before the search finds it. */
const INDEXING_LIMIT_MS = 90_000

export class SearchDriver {
  private query = ''
  private readonly palette: Palette
  private readonly found: SearchResults

  constructor(private readonly scenario: Scenario) {
    this.palette = new Palette(scenario)
    this.found = new SearchResults(scenario)
  }

  private async type(): Promise<void> {
    const { page } = this.scenario
    const field = this.palette.searchField()
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
    return this.found.results(text, group)
  }

  private async waitForResult(text: string, group: string): Promise<Locator> {
    // A new file is found once indexed, so the words are typed again until the result shows.
    const results = this.getResults(text, group)
    await expect(async () => {
      await this.type()
      await expect(results.first()).toBeVisible({ timeout: 3_000 })
    }).toPass({ timeout: INDEXING_LIMIT_MS })
    return this.found.pickRow(results, text)
  }

  async expectResult(text: string, group: string): Promise<void> {
    await this.waitForResult(text, group)
  }

  async expectNoResult(text: string, group: string): Promise<void> {
    const results = this.getResults(text, group)
    if (this.query === '') {
      await expect(results).toHaveCount(0)
      return
    }
    // The index removes a file that lost its text after a delay, so the words are typed again until the index has no hit for the file.
    await expect(async () => {
      await this.type()
      await expect(results).toHaveCount(0, { timeout: 3_000 })
    }).toPass({ timeout: INDEXING_LIMIT_MS })
  }

  async chooseResult(text: string, group: string): Promise<void> {
    await (await this.waitForResult(text, group)).click()
  }

  /** Lights the result with the arrow keys and opens the list of the actions it offers. */
  async openActions(text: string, group: string): Promise<void> {
    const { page } = this.scenario
    const row = await this.waitForResult(text, group)
    // The groups answer in any order, and one that answers can move the light back to the first row.
    await expect(this.found.busyGroups()).toHaveCount(0)
    const rows = this.found.rows()
    for (let step = 0; step < (await rows.count()) && (await row.getAttribute('aria-selected')) !== 'true'; step += 1) {
      await page.keyboard.press('ArrowDown')
    }
    await expect(row).toHaveAttribute('aria-selected', 'true')
    await page.keyboard.press('Control+k')
    await expect(this.found.actionRows().first()).toBeVisible()
  }

  async listActions(text: string, group: string): Promise<string[]> {
    await this.openActions(text, group)
    const names = await this.found.readActionNames()
    await this.scenario.page.keyboard.press('Escape')
    return names
  }

  /** The actions of the result are these and no other, include these, or include none of these. */
  async expectActions(
    text: string,
    group: string,
    rows: readonly (readonly string[])[],
    rule: 'only' | 'include' | 'exclude',
  ): Promise<void> {
    const wanted = parseColumn(rows, 'action')
    const names = await this.listActions(text, group)
    if (rule === 'only') expect(names).toStrictEqual(wanted)
    else if (rule === 'include') expect(names).toEqual(expect.arrayContaining(wanted))
    else expect(names.filter((name) => wanted.includes(name))).toStrictEqual([])
  }

  async chooseAction(action: string): Promise<void> {
    const rows = this.found.actionNamed(action)
    await expect(rows, `the result offers no action "${action}"`).toHaveCount(1)
    await rows.click()
  }
}
