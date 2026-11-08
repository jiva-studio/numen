/** The person at a tab, whatever it holds: closing it, finding the one in front, its position, what it shows. */
import { expect, type Locator } from 'playwright/test'
import { getNoteTab, getTabName, revealTreeRow } from '../../../e2e/desktop/tests/support/vault.js'
import { assertSingleTab, getAddressTitle, getEscaped, getStem, getTitlePattern, parseReference } from './rules.js'
import type { Scenario } from './scenario.js'
import { expectTabHighlight } from './tab-highlight.js'
import { expectTabAt, openTabAt } from './tab-position.js'

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
    return this.scenario.page.getByRole('tabpanel', {
      name: getTitlePattern(this.getTitle(reference, name)),
    })
  }

  async close(reference: string, name: string): Promise<void> {
    const tab = await this.getTab(reference, name)
    await tab.getByRole('button', { name: /^Close / }).click()
    await expect(tab).toHaveCount(0)
  }

  async expectOpen(reference: string, name: string): Promise<void> {
    await expect(await this.getTab(reference, name)).toBeVisible()
  }

  async expectHasText(reference: string, name: string, text: string): Promise<void> {
    const panel = this.getPanel(reference, name)
    await expect(panel).toBeVisible()
    await expect(panel).toContainText(text)
  }

  async expectDoesNotHaveText(reference: string, name: string, text: string): Promise<void> {
    const panel = this.getPanel(reference, name)
    await expect(panel).toBeVisible()
    await expect(panel).not.toContainText(text)
  }

  async openFile(filePath: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await (await revealTreeRow(this.scenario.page, filePath)).press('Enter')
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
    await expect(await this.getTab(reference, name)).toHaveAttribute('aria-selected', 'true')
  }

  async openAt(reference: string, name: string, position: string, where: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await openTabAt(this.scenario.page, await this.scenario.getPath(reference, name), position, where)
  }

  async expectAt(reference: string, name: string, position: string, where: string): Promise<void> {
    await expectTabAt(this.scenario.page, await this.scenario.getPath(reference, name), position, where)
  }

  async expectHighlight(reference: string, name: string, text: string): Promise<void> {
    await expectTabHighlight(this.scenario.page, await this.scenario.getPath(reference, name), text)
  }
}
