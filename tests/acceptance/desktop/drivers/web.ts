/** The web as the person's vault reaches it: the addresses the stand-in server answers, and with what. */
import { removePageText, writePageText } from '../../../e2e/desktop/tests/support/media.js'

export class WebDriver {
  private readonly addresses = new Set<string>()

  setPageText(address: string, text: string): void {
    writePageText(address, text)
    this.addresses.add(address)
  }

  removePageTexts(): void {
    for (const address of this.addresses) removePageText(address)
    this.addresses.clear()
  }
}
