/** The person at the Settings tab: the settings of the window and of each plugin, read and changed. */
import { expect, type Locator } from 'playwright/test'
import type { ApplicationDriver } from './application.js'
import { SettingsPage } from './components/SettingsPage.js'
import { escapeRegExp } from './rules.js'
import type { Scenario } from './scenario.js'

const OPEN_SETTINGS = 'Open settings'

export class SettingsDriver {
  constructor(
    private readonly scenario: Scenario,
    private readonly application: ApplicationDriver,
  ) {}

  private get settings(): SettingsPage {
    return new SettingsPage(this.scenario.page)
  }

  private async showPage(plugin?: string): Promise<Locator> {
    await this.scenario.ensureWorkspace()
    const tab = this.settings.getTab()
    if ((await tab.count()) === 0) await this.application.runCommand(OPEN_SETTINGS)
    else await tab.click()
    return this.settings.showPage(plugin)
  }

  async set(name: string, value: string, plugin?: string): Promise<void> {
    const row = await this.settings.requireRow(await this.showPage(plugin), name)
    const choice = this.settings.getChoice(row)
    if ((await choice.count()) === 0) {
      const input = this.settings.getTextInput(row)
      await expect(input, `the setting "${name}" is not a choice and not a text`).toHaveCount(1)
      await input.fill(value)
      await input.press('Tab')
      return
    }
    await choice.click()
    const option = this.settings.getOption(value)
    await expect(option, `the setting "${name}" has no option "${value}"`).toBeVisible()
    await option.click()
  }

  async expectValue(name: string, value: string, plugin?: string): Promise<void> {
    const row = await this.settings.requireRow(await this.showPage(plugin), name)
    const choice = this.settings.getChoice(row)
    if ((await choice.count()) > 0) await expect(choice).toHaveText(value)
    else await expect(this.settings.getTextInput(row)).toHaveValue(value)
  }

  private async countOptions(row: Locator, option: string): Promise<number> {
    const pattern = new RegExp(escapeRegExp(option))
    const choice = this.settings.getChoice(row)
    if ((await choice.count()) === 0) {
      const fixed = this.settings.getFixedValue(row)
      await expect(fixed, 'the setting has neither a menu of options nor one fixed option').toHaveCount(1)
      return pattern.test(await fixed.innerText()) ? 1 : 0
    }
    await choice.click()
    await expect(this.settings.getAnyOption()).toBeVisible()
    const count = await this.settings.getOptions(pattern).count()
    await this.scenario.page.keyboard.press('Escape')
    return count
  }

  async expectOption(name: string, option: string, has: boolean, plugin?: string): Promise<void> {
    const row = await this.settings.requireRow(await this.showPage(plugin), name)
    await expect.poll(() => this.countOptions(row, option)).toBe(has ? 1 : 0)
  }

  async expectField(plugin: string, name: string, has: boolean): Promise<void> {
    const row = this.settings.getRow(await this.showPage(plugin), name)
    await expect(row).toHaveCount(has ? 1 : 0)
  }
}
