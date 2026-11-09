/** The web as the person's vault reaches it: the addresses the stand-in server answers, and with what. */
import { removePageText, writePageText } from '../../../e2e/desktop/tests/support/media.js'
import type { Scenario } from './scenario.js'

export class WebDriver {
  private readonly addresses = new Set<string>()

  constructor(private readonly scenario: Scenario) {}

  /** Gives the page of an address a text: in a world, to the server of the world; else to the shared stand-in. */
  setPageText(address: string, text: string): void {
    if (this.scenario.hasWorld) {
      this.scenario.getWorld().setPageText(address, text)
      return
    }
    writePageText(address, text)
    this.addresses.add(address)
  }

  removePageTexts(): void {
    for (const address of this.addresses) removePageText(address)
    this.addresses.clear()
  }
}
