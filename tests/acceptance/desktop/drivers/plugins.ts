/** The person at the settings of the plugins: turning one off and, when the scenario ends, on again. */
import { expect, type Locator } from 'playwright/test'
import { SettingsPage } from './components/SettingsPage.js'
import type { Scenario } from './scenario.js'

export class PluginDriver {
  private readonly disabled = new Set<string>()

  constructor(private readonly scenario: Scenario) {}

  async disable(name: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    this.disabled.add(name)
    await this.setEnabled(name, false)
  }

  async enable(name: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await this.setEnabled(name, true)
    this.disabled.delete(name)
  }

  async expectEnabled(name: string, isEnabled: boolean): Promise<void> {
    await this.scenario.ensureWorkspace()
    const toggle = await this.showToggle(name)
    await expect(toggle).toBeChecked({ checked: isEnabled })
  }

  async enableDisabled(): Promise<void> {
    for (const name of this.disabled) await this.setEnabled(name, true)
    this.disabled.clear()
  }

  private async setEnabled(name: string, isEnabled: boolean): Promise<void> {
    const toggle = await this.showToggle(name)
    // The switch moves when the core has answered its write, so the click is not waited for to change it.
    if ((await toggle.isChecked()) !== isEnabled) await toggle.click()
    await expect(toggle).toBeChecked({ checked: isEnabled })
  }

  /** Shows the switch of the plugin in the Settings tab. */
  private async showToggle(name: string): Promise<Locator> {
    const { page } = this.scenario
    const field = page.getByRole('combobox', { name: 'Search the vault' })
    for (let press = 0; press < 2 && (await field.isVisible()); press += 1) await page.keyboard.press('Escape')
    await expect(field).toBeHidden()
    await page.keyboard.press('Control+p')
    await field.fill('Open settings')
    await page.getByRole('option', { name: /^Open settings$/ }).first().click()
    const settingsPage = new SettingsPage(page)
    const settings = settingsPage.getPanel()
    const toggle = settingsPage.getPluginSwitch(settings, name)
    const plugin = settingsPage.getPluginButton(settings, name)
    // The settings open on the page of the plugin shown last, which may be another plugin.
    await expect(settings).toBeVisible()
    await expect(toggle.or(plugin).or(settingsPage.getBackButton()).first()).toBeVisible()
    if (!(await toggle.isVisible()) && !(await plugin.isVisible())) {
      await settingsPage.getBackButton().click()
    }
    if (!(await toggle.isVisible())) await plugin.click()
    await expect(toggle).toBeVisible()
    return toggle
  }
}
