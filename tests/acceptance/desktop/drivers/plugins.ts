/** The person at the settings of the plugins: turning one off and, when the scenario ends, on again. */
import { expect } from 'playwright/test'
import type { Scenario } from './scenario.js'

export class PluginDriver {
  private readonly disabled = new Set<string>()

  constructor(private readonly scenario: Scenario) {}

  async disable(name: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    this.disabled.add(name)
    await this.setEnabled(name, false)
  }

  async enableDisabled(): Promise<void> {
    for (const name of this.disabled) await this.setEnabled(name, true)
    this.disabled.clear()
  }

  private async setEnabled(name: string, isEnabled: boolean): Promise<void> {
    const { page } = this.scenario
    const field = page.getByRole('combobox', { name: 'Search the vault' })
    for (let press = 0; press < 2 && (await field.isVisible()); press += 1) await page.keyboard.press('Escape')
    await expect(field).toBeHidden()
    await page.keyboard.press('Control+p')
    await field.fill('Open settings')
    await page.getByRole('option', { name: /^Open settings$/ }).first().click()
    const settings = page.getByRole('tabpanel', { name: /Settings/ })
    const toggle = settings.getByRole('switch', { name: new RegExp(`Enabled ${name}`) })
    const plugin = settings.getByRole('button', { name: new RegExp(name) })
    // The settings open on the page of the plugin shown last.
    await expect(toggle.or(plugin)).toBeVisible()
    if (!(await toggle.isVisible())) await plugin.click()
    // The switch moves when the core has answered its write, so the click is not waited for to change it.
    if ((await toggle.isChecked()) !== isEnabled) await toggle.click()
    await expect(toggle).toBeChecked({ checked: isEnabled })
  }
}
