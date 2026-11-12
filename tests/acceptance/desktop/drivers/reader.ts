/** The person at a reader and at the panes: pages and their buttons, the contents of a book, the place of a tab. */
import { expect, type Locator } from 'playwright/test'
import { readBox } from '../../../e2e/desktop/tests/support/browser.js'
import { escapeRegExp, getStem, parseColumn, parseReference, type Place, type TabKindChoice } from './rules.js'
import { Panes } from './components/Panes.js'
import { ReaderView } from './components/ReaderView.js'
import type { Scenario } from './scenario.js'
import type { TabDriver } from './tabs.js'

export class ReaderDriver {
  constructor(
    private readonly scenario: Scenario,
    private readonly tab: TabDriver,
  ) {}

  private async getView(kind: TabKindChoice, reference: string, name: string): Promise<ReaderView> {
    return new ReaderView(await this.tab.getTabPanel(kind, reference, name))
  }

  private get panes(): Panes {
    return new Panes(this.scenario.page)
  }

  async goToPage(place: Place, kind: TabKindChoice, reference: string, name: string): Promise<void> {
    const page = place.where
    if (place.position !== 'page' || !/^\d+$/.test(page)) throw new Error(`"${place.position} ${page}" is not a page: write page "12"`)
    const field = (await this.getView(kind, reference, name)).getPageField()
    await field.fill(page)
    await field.press('Enter')
    await expect(field).toHaveValue(page)
  }

  private async getButton(label: string, kind: TabKindChoice, reference: string, name: string): Promise<Locator> {
    return (await this.getView(kind, reference, name)).requireButton(label)
  }

  async clickButton(label: string, kind: TabKindChoice, reference: string, name: string): Promise<void> {
    await (await this.getButton(label, kind, reference, name)).click()
  }

  async expectButtonEnabled(label: string, kind: TabKindChoice, reference: string, name: string): Promise<void> {
    await expect(await this.getButton(label, kind, reference, name)).toBeEnabled()
  }

  async expectButtonDisabled(label: string, kind: TabKindChoice, reference: string, name: string): Promise<void> {
    await expect(await this.getButton(label, kind, reference, name)).toBeDisabled()
  }

  async expectContents(rows: readonly (readonly string[])[], kind: TabKindChoice, reference: string, name: string): Promise<void> {
    const entries = (await this.getView(kind, reference, name)).getChapters()
    await expect(entries).toHaveText(parseColumn(rows, 'chapter'))
  }

  async expectContentsHidden(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    await expect((await this.getView(kind, reference, name)).getContents()).toBeHidden()
  }

  async chooseChapter(chapter: string, kind: TabKindChoice, reference: string, name: string): Promise<void> {
    const entry = (await this.getView(kind, reference, name)).getChapter(chapter)
    await expect(entry, `the contents have no chapter "${chapter}"`).toBeVisible()
    await entry.click()
  }

  private getTabName(reference: string, name: string): RegExp {
    const title = parseReference(reference, name).reference === 'at' ? getStem(name) : name
    return new RegExp(`^(Linked tab )?${escapeRegExp(title)}`, 'i')
  }

  async moveToOtherPane(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    await this.tab.getTabPanel(kind, reference, name)
    const { page } = this.scenario
    const tabName = this.getTabName(reference, name)
    const panes = await this.panes.listTwo()
    const holders = await Promise.all(panes.map(async (one) => (await this.panes.getTabIn(one, tabName).count()) > 0))
    const from = holders.indexOf(true)
    if (from < 0) throw new Error(`no pane holds the tab "${name}"`)
    const source = panes[from]
    const target = panes[1 - from]
    const start = await readBox(this.panes.getTabIn(source, tabName))
    const end = await readBox(target)
    await page.mouse.move(start.x + start.width / 2, start.y + start.height / 2)
    await page.mouse.down()
    await page.mouse.move(end.x + end.width / 2, end.y + end.height / 2, { steps: 15 })
    await page.mouse.up()
    await expect(this.panes.getTabIn(target, tabName)).toHaveCount(1)
    await expect(this.panes.getTabIn(source, tabName)).toHaveCount(0)
  }

  async expectOwnPane(kind: TabKindChoice, reference: string, name: string): Promise<void> {
    await this.tab.getTabPanel(kind, reference, name)
    const tabName = this.getTabName(reference, name)
    const panes = this.panes.getPaneWithTab(tabName)
    await expect(panes, `the tab "${name}" is not in one pane`).toHaveCount(1)
    await expect(this.panes.getTabs(panes.first())).toHaveCount(1)
  }
}
