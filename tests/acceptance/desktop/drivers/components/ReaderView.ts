/** A page or book reader inside the panel of a tab: the page field, the buttons, the contents. */
import { expect, type Locator } from 'playwright/test'

export class ReaderView {
  constructor(private readonly panel: Locator) {}

  getPageField(): Locator {
    return this.panel.getByRole('spinbutton', { name: 'Page' })
  }

  /** The one button with the label; the view has no other. */
  async requireButton(label: string): Promise<Locator> {
    const button = this.panel.getByRole('button', { name: label, exact: true })
    await expect(button, `the tab has no button "${label}"`).toHaveCount(1)
    return button
  }

  getContents(): Locator {
    return this.panel.getByRole('group', { name: 'Contents' })
  }

  getChapters(): Locator {
    return this.getContents().getByRole('option')
  }

  getChapter(chapter: string): Locator {
    return this.getContents().getByRole('option', { name: chapter, exact: true })
  }
}
