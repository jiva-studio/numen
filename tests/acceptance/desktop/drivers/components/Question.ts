/** A question over the window: its text, and the buttons that answer it. */
import { expect, type Locator, type Page } from 'playwright/test'

export class Question {
  constructor(private readonly page: Page) {}

  getRoot(): Locator {
    return this.page.getByRole('alertdialog')
  }

  /** Waits for the question whose text matches, presses the answer and waits for the question to close. */
  async answer(button: string, text: string | RegExp): Promise<void> {
    await expect(this.getRoot()).toHaveAccessibleDescription(text)
    await this.getRoot().getByRole('button', { name: button, exact: true }).click()
    await expect(this.getRoot()).toHaveCount(0)
  }
}
