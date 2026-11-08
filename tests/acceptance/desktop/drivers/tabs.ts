/** The person at a tab, whatever it holds: closing it, finding the one in front, its place, what it shows. */
import { expect, type Locator } from 'playwright/test'
import { getNoteTab, getTabName, revealTreeRow } from '../../../e2e/desktop/tests/support/vault.js'
import { assertSingleTab, getEscaped, getStem, parseReference } from './rules.js'
import type { Scenario } from './scenario.js'
import { expectTabAt, openTabAt } from './tab-position.js'

export class TabDriver {
  constructor(private readonly scenario: Scenario) {}

  private async getTab(how: string, value: string): Promise<Locator> {
    const { page } = this.scenario
    if (parseReference(how, value).how === 'at') {
      if (value.endsWith('.md')) return getNoteTab(page, value)
      return page.getByRole('tab', { name: new RegExp(`^${getEscaped(getStem(value))}`, 'i') })
    }
    const tabs = page.getByRole('tab', { name: new RegExp(`^${getEscaped(value)}( Close |$)`) })
    await expect(tabs.first()).toBeVisible()
    assertSingleTab(value, await tabs.count())
    return tabs
  }

  private getTitle(how: string, value: string): string {
    if (parseReference(how, value).how === 'titled') return value
    return value.endsWith('.md') ? getTabName(value) : getStem(value)
  }

  private getPanel(how: string, value: string): Locator {
    return this.scenario.page.getByRole('tabpanel', {
      name: new RegExp(`^${getEscaped(this.getTitle(how, value))}( Close |$)`),
    })
  }

  async close(how: string, value: string): Promise<void> {
    const tab = await this.getTab(how, value)
    const name = this.getTitle(how, value)
    await this.scenario.page.getByRole('button', { name: `Close ${name}`, exact: true }).click()
    await expect(tab).toHaveCount(0)
  }

  async expectOpen(how: string, value: string): Promise<void> {
    await expect(await this.getTab(how, value)).toBeVisible()
  }

  async expectShows(how: string, value: string, shows: boolean, text: string): Promise<void> {
    const panel = this.getPanel(how, value)
    await expect(panel).toBeVisible()
    if (shows) await expect(panel).toContainText(text)
    else await expect(panel).not.toContainText(text)
  }

  async openFile(filePath: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await (await revealTreeRow(this.scenario.page, filePath)).press('Enter')
  }

  async playRecording(how: string, value: string): Promise<void> {
    await this.getPanel(how, value).getByRole('button', { name: /^Play/ }).click()
  }

  async expectPlaying(how: string, value: string): Promise<void> {
    const panel = this.getPanel(how, value)
    await expect(panel.getByRole('button', { name: /^Pause/ })).toBeVisible()
    await expect(panel).toContainText(/0:0[1-9]/)
  }

  async expectActive(how: string, value: string): Promise<void> {
    await expect(await this.getTab(how, value)).toHaveAttribute('aria-selected', 'true')
  }

  async openAt(how: string, value: string, place: string, at: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await openTabAt(this.scenario.page, await this.scenario.getPath(how, value), place, at)
  }

  async expectAt(how: string, value: string, place: string, at: string): Promise<void> {
    await expectTabAt(this.scenario.page, await this.scenario.getPath(how, value), place, at)
  }
}
