/** The palette and the search of the vault: the field, the rows, the groups and the question it asks. */
import { expect, type Locator } from 'playwright/test'
import type { Host } from './host.js'

export class Palette {
  constructor(private readonly host: Host) {}

  /** The field the palette is typed into, found by its marker. */
  field(): Locator {
    return this.host.page.locator('input[data-palette="field"]').first()
  }

  /** The field the search of the vault is typed into, found by its role and name. */
  searchField(): Locator {
    return this.host.page.getByRole('combobox', { name: 'Search the vault' })
  }

  /** The text field that asks for a name inside the file tree. */
  treeField(): Locator {
    return this.host.page.getByRole('tree').getByRole('textbox')
  }

  /** The command the open palette has taken, shown before its question. */
  crumb(): Locator {
    return this.host.page.locator('[data-palette="crumb"]')
  }

  /** The name cell of the rows whose name matches the pattern. */
  nameCell(pattern: RegExp): Locator {
    return this.host.page.locator('[data-palette="name"]', { hasText: pattern })
  }

  /** All rows of the open palette. */
  rows(): Locator {
    return this.host.page.getByRole('option')
  }

  /** The rows whose name matches the pattern. */
  rowNamed(pattern: RegExp): Locator {
    return this.rows().filter({ has: this.nameCell(pattern) })
  }

  /** The row whose accessible name matches the pattern. */
  optionNamed(name: RegExp): Locator {
    return this.host.page.getByRole('option', { name })
  }

  /** The titles of the groups the palette lists. */
  async readGroupTitles(): Promise<string[]> {
    const titles = await this.host.page.locator('[data-palette="panel"] [role="group"] [data-palette="title"]').allTextContents()
    return titles.map((one) => one.trim())
  }

  /** The groups of the open list that have not answered yet. */
  busyGroups(): Locator {
    return this.host.page.getByRole('listbox').locator('[role="group"][aria-busy="true"]')
  }

  /** Closes the palette or the search when it is open, as it covers the window and takes the clicks. */
  async close(): Promise<void> {
    const field = this.field()
    for (let attempt = 0; attempt < 3 && (await field.isVisible()); attempt += 1) {
      await this.host.page.keyboard.press('Escape')
      await this.host.page.waitForTimeout(100)
    }
    await expect(field).toBeHidden()
  }
}
