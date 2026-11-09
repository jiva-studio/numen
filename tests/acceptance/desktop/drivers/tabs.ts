/** The person at a tab, whatever it holds: closing it, finding the one in front, its position, what it shows. */
import { expect, type Locator } from 'playwright/test'
import { readActivePath } from '../../../e2e/desktop/tests/support/test.js'
import { getNoteTab, getTabName, revealTreeRow } from '../../../e2e/desktop/tests/support/vault.js'
import { assertSingleTab, getAddressTitle, getEscaped, getStem, getTitlePattern, parseReference } from './rules.js'
import type { Scenario } from './scenario.js'
import { expectTabHighlight } from './tab-highlight.js'
import { readOpenTabs } from '../../../e2e/desktop/tests/support/layout.js'
import { expectTabAt, openTabAt } from './tab-position.js'

/** The tab of the file tree: it shows each file and folder as a row named by its path. */
const FILES_TITLE = 'Files'

export class TabDriver {
  constructor(private readonly scenario: Scenario) {}

  private async getTab(reference: string, name: string): Promise<Locator> {
    const { page } = this.scenario
    if (parseReference(reference, name).reference === 'at') {
      if (name.endsWith('.md')) return getNoteTab(page, name)
      return page.getByRole('tab', { name: new RegExp(`^${getEscaped(getStem(name))}`, 'i') })
    }
    const tabs = page.getByRole('tab', { name: getTitlePattern(this.getTitle(reference, name)) })
    await expect(tabs.first()).toBeVisible()
    assertSingleTab(name, await tabs.count())
    return tabs
  }

  private getTitle(reference: string, name: string): string {
    const kind = parseReference(reference, name).reference
    if (kind === 'titled') return name
    if (kind === 'for the web address') return getAddressTitle(name)
    return name.endsWith('.md') ? getTabName(name) : getStem(name)
  }

  private getPanel(reference: string, name: string): Locator {
    const title = this.getTitle(reference, name)
    // The panel of a file that is not a note is named by the file name, with its extension.
    const isFile = parseReference(reference, name).reference === 'at' && !name.endsWith('.md')
    const pattern = isFile ? new RegExp(`^(Linked tab )?${getEscaped(title)}(\\.\\w+)?( Close |$)`) : getTitlePattern(title)
    return this.scenario.page.getByRole('tabpanel', { name: pattern })
  }

  async close(reference: string, name: string): Promise<void> {
    const tab = await this.getTab(reference, name)
    await tab.getByRole('button', { name: /^Close / }).click()
    await expect(tab).toHaveCount(0)
  }

  private isNote(reference: string, name: string): boolean {
    return parseReference(reference, name).reference === 'at' && name.endsWith('.md')
  }

  private async listNotePaths(): Promise<string[]> {
    const { tabs } = await readOpenTabs(this.scenario.page.request)
    return tabs.filter((one) => one.kind === 'note').map((one) => one.path)
  }

  async expectOpen(reference: string, name: string): Promise<void> {
    if (this.isNote(reference, name)) await expect.poll(() => this.listNotePaths()).toContain(name)
    await expect(await this.getTab(reference, name)).toBeVisible()
  }

  async expectNotOpen(reference: string, name: string): Promise<void> {
    if (this.isNote(reference, name)) {
      await expect.poll(() => this.listNotePaths()).not.toContain(name)
    } else {
      await expect(this.scenario.page.getByRole('tab', { name: getTitlePattern(this.getTitle(reference, name)) })).toHaveCount(0)
    }
  }

  async activate(reference: string, name: string): Promise<void> {
    await (await this.getTab(reference, name)).click()
  }

  async expectHasText(reference: string, name: string, text: string): Promise<void> {
    const panel = this.getPanel(reference, name)
    await expect(panel).toBeVisible()
    if (this.getTitle(reference, name) === FILES_TITLE) {
      await expect(panel.locator(`[data-tree-row="${text}"]`)).toBeVisible()
      return
    }
    await expect(panel).toContainText(text)
  }

  async expectDoesNotHaveText(reference: string, name: string, text: string): Promise<void> {
    const panel = this.getPanel(reference, name)
    await expect(panel).toBeVisible()
    await expect(panel).not.toContainText(text)
  }

  async openFile(filePath: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    this.scenario.rememberFile(filePath)
    await (await revealTreeRow(this.scenario.page, filePath)).press('Enter')
    await expect.poll(() => readActivePath(this.scenario.page.request)).toBe(filePath)
  }

  async playRecording(reference: string, name: string): Promise<void> {
    await this.getPanel(reference, name).getByRole('button', { name: /^Play/ }).click()
  }

  async expectPlaying(reference: string, name: string): Promise<void> {
    const panel = this.getPanel(reference, name)
    await expect(panel.getByRole('button', { name: /^Pause/ })).toBeVisible()
    await expect(panel).toContainText(/0:0[1-9]/)
  }

  async expectActive(reference: string, name: string): Promise<void> {
    if (parseReference(reference, name).reference === 'at' && name.endsWith('.md')) {
      // A tab linked to another tab keeps its own id and shows the note it was sent to.
      await expect.poll(() => readActivePath(this.scenario.page.request)).toBe(name)
      return
    }
    await expect(await this.getTab(reference, name)).toHaveAttribute('aria-selected', 'true')
  }

  async openAt(reference: string, name: string, position: string, where: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await openTabAt(this.scenario.page, await this.scenario.getPath(reference, name), position, where)
  }

  async expectAt(reference: string, name: string, position: string, where: string): Promise<void> {
    await expectTabAt(this.scenario.page, await this.scenario.getPath(reference, name), position, where)
  }

  async expectHighlight(reference: string, name: string, text?: string): Promise<void> {
    await expectTabHighlight(this.scenario.page, await this.scenario.getPath(reference, name), text)
  }
}
