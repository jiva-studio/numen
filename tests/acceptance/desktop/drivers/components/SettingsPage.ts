/** The Settings tab: the pages of the window and of each plugin, their rows, choices, texts and switches. */
import { expect, type Locator, type Page } from 'playwright/test'
import { escapeRegExp } from '../rules.js'

const SYSTEM = /^System/
const BACK = /All plugins/i

export class SettingsPage {
  constructor(private readonly page: Page) {}

  getTab(): Locator {
    return this.page.getByRole('tab', { name: /^Settings/ })
  }

  getPanel(): Locator {
    return this.page.getByRole('tabpanel', { name: /Settings/ })
  }

  getBackButton(): Locator {
    return this.getPanel().getByRole('button', { name: BACK })
  }

  /** The panel on the page of the plugin, or on the page of the window when no plugin is named; the tab is open already. */
  async showPage(plugin?: string): Promise<Locator> {
    const panel = this.getPanel()
    await expect(panel).toBeVisible()
    const entry = plugin === undefined ? SYSTEM : new RegExp(escapeRegExp(plugin), 'i')
    const button = panel.getByRole('button', { name: entry }).first()
    // A page of one plugin hides the list of the pages: the way to another is back to the list.
    if (!(await button.isVisible())) await panel.getByRole('button', { name: BACK }).click()
    await button.click()
    return panel
  }

  getRow(panel: Locator, name: string): Locator {
    return panel
      .locator('.settings__row')
      .filter({ has: this.page.locator('.settings__name', { hasText: new RegExp(`^${escapeRegExp(name)}$`) }) })
  }

  async requireRow(panel: Locator, name: string): Promise<Locator> {
    const row = this.getRow(panel, name)
    await expect(row, `the page has no setting "${name}"`).toHaveCount(1)
    return row
  }

  getChoice(row: Locator): Locator {
    return row.locator('.settings__choice')
  }

  getTextInput(row: Locator): Locator {
    return row.locator('.settings__text-input')
  }

  getFixedValue(row: Locator): Locator {
    return row.locator('.settings__fixed')
  }

  /** The option of an open menu whose name starts with the value. */
  getOption(value: string): Locator {
    return this.page.getByRole('menuitemradio', { name: new RegExp(`^${escapeRegExp(value)}`) }).first()
  }

  getAnyOption(): Locator {
    return this.page.getByRole('menuitemradio').first()
  }

  getOptions(pattern: RegExp): Locator {
    return this.page.getByRole('menuitemradio', { name: pattern })
  }

  getPluginSwitch(panel: Locator, plugin: string): Locator {
    return panel.getByRole('switch', { name: new RegExp(`Enabled ${plugin}`) })
  }

  getPluginButton(panel: Locator, plugin: string): Locator {
    return panel.getByRole('button', { name: new RegExp(plugin) })
  }
}
