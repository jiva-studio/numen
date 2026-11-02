/**
 * What every list that is searched looks like in the window: one container with the search line
 * as its first child and the rows in the same container. A popup built any other way is a second
 * component, and the specs of every consumer assert this so that they cannot drift apart.
 */
import { expect, type Locator, type Page } from 'playwright/test'

/** Fails unless the scope holds one searchable list with its search line first and its rows inside. */
export const expectSearchableList = async (scope: Page | Locator): Promise<void> => {
  const frame = scope.locator('[data-numen-searchable-list]')
  await expect(frame).toHaveCount(1)
  await expect(frame.locator(':scope > input[role="combobox"]:first-child')).toHaveCount(1)
  await expect(frame.getByRole('listbox')).toHaveCount(1)
}
