/** The file tree of the Files tab: its rows, their folding and the menu of a row. */
import { expect, type Locator, type Page } from 'playwright/test'

export class FileTree {
  constructor(private readonly page: Page) {}

  async showTab(): Promise<void> {
    await this.page.getByRole('tab', { name: /^Files/ }).click()
  }

  getRow(entryPath: string): Locator {
    return this.page.locator(`[data-tree-row="${entryPath}"]`)
  }

  /** Unfolds the folder row, which Enter folds when it is open already. */
  async expand(row: Locator): Promise<void> {
    if ((await row.getAttribute('aria-expanded')) !== 'true') await row.press('Enter')
    await expect(row).toHaveAttribute('aria-expanded', 'true')
  }

  /** Unfolds the folder at the path once its row is shown. */
  async expandFolder(folderPath: string): Promise<void> {
    const parent = this.getRow(folderPath)
    await expect(parent).toBeVisible()
    await this.expand(parent)
  }

  async openMenu(row: Locator): Promise<void> {
    await row.click({ button: 'right' })
  }

  getMenuItem(label: string): Locator {
    return this.page.getByRole('menuitem', { name: label, exact: true })
  }

  getFirstMenuItem(): Locator {
    return this.page.getByRole('menuitem').first()
  }

  /** The menu item that opens a submenu. */
  getSubmenuItem(): Locator {
    return this.page.getByRole('menuitem').and(this.page.locator('[aria-haspopup="menu"]'))
  }
}
