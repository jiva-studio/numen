/** The person at a tab, whatever it holds: closing it, finding the one in front, its place. */
import { expect, type Locator } from 'playwright/test'
import {
  getNoteTab,
  getTabName,
} from '../../../e2e/desktop/tests/support/vault.js'
import type { Scenario } from './scenario.js'
import { expectTabAt, getEscaped, getStem, openTabAt } from './tab-position.js'

export class TabDriver {
  constructor(private readonly scenario: Scenario) {}

  private getTab(notePath: string): Locator {
    if (notePath.endsWith('.md')) return getNoteTab(this.scenario.page, notePath)
    return this.scenario.page.getByRole('tab', { name: new RegExp(`^${getEscaped(getStem(notePath))}`, 'i') })
  }

  async close(how: string, value: string): Promise<void> {
    const notePath = this.scenario.getPath(how, value)
    const name = notePath.endsWith('.md') ? getTabName(notePath) : getStem(notePath)
    await this.scenario.page.getByRole('button', { name: `Close ${name}`, exact: true }).click()
    await expect(this.getTab(notePath)).toHaveCount(0)
  }

  async expectActive(how: string, value: string): Promise<void> {
    const tab = this.getTab(this.scenario.getPath(how, value))
    await expect(tab).toHaveAttribute('aria-selected', 'true')
  }

  async openAt(how: string, value: string, place: string, at: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await openTabAt(this.scenario.page, this.scenario.getPath(how, value), place, at)
  }

  async expectAt(how: string, value: string, place: string, at: string): Promise<void> {
    await expectTabAt(this.scenario.page, this.scenario.getPath(how, value), place, at)
  }
}
