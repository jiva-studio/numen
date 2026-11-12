/** The person at the cards of a Deck tab: the value of a field of a card. */
import { expect, type Locator } from 'playwright/test'
import { getCardFile } from './cards.js'
import { DEFAULT_VAULT, parseReference, type TabRef } from './rules.js'
import type { Scenario } from './scenario.js'
import type { TabDriver } from './tabs.js'

export class DeckDriver {
  constructor(
    private readonly scenario: Scenario,
    private readonly tab: TabDriver,
  ) {}

  private async getField(field: string, card: string, tab: TabRef): Promise<Locator> {
    const panel = await this.tab.getTabPanel(tab.kind, tab.reference, tab.name)
    const cardPanel = panel.getByRole('article', { name: card, exact: true })
    await expect(cardPanel, `the tab has no card "${card}"`).toHaveCount(1)
    const input = cardPanel.getByRole('textbox', { name: field, exact: true })
    await expect(input, `the card "${card}" has no field "${field}"`).toHaveCount(1)
    return input
  }

  async write(value: string, field: string, card: string, tab: TabRef): Promise<void> {
    await (await this.getField(field, card, tab)).fill(value)
    await this.expectSaved(value, tab)
  }

  private async expectSaved(value: string, tab: TabRef): Promise<void> {
    const world = this.scenario.getWorld()
    const file = parseReference(tab.reference, tab.name).reference === 'at' ? tab.name : getCardFile(tab.name)
    await expect.poll(() => world.readText(this.scenario.getVault(DEFAULT_VAULT), file)).toContain(value)
  }

  async expectValue(value: string, field: string, card: string, tab: TabRef): Promise<void> {
    await expect(await this.getField(field, card, tab)).toHaveValue(value)
  }
}
