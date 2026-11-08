/** The person at a tab, whatever it holds: closing it, finding the one in front, its place. */
import { expect, type Locator } from 'playwright/test'
import { getNoteTab, getTabName } from '../../../e2e/desktop/tests/support/vault.js'
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

  async close(how: string, value: string): Promise<void> {
    const tab = await this.getTab(how, value)
    const name = how === 'titled' ? value : value.endsWith('.md') ? getTabName(value) : getStem(value)
    await this.scenario.page.getByRole('button', { name: `Close ${name}`, exact: true }).click()
    await expect(tab).toHaveCount(0)
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
