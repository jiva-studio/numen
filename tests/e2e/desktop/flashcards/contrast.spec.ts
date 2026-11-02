import path from 'node:path'
import { expect, test, type Page } from 'playwright/test'
import { expectReadable } from '../tests/support/contrast.js'

const vault = process.env.NUMEN_FLASHCARDS_VAULT ?? ''

/** Opens the decks of the run's vault. */
async function openDecks(page: Page): Promise<void> {
  await page.goto('/')
  await page.getByRole('button', { name: new RegExp(path.basename(vault)) }).click()
  await expect(page.getByRole('heading', { name: path.basename(vault) })).toBeVisible()
}

/**
 * The primary button of the decks and every button of a card under review, in both halves of the
 * theme: the Review button is drawn on the accent, and its text must be readable there.
 */
for (const scheme of ['light', 'dark'] as const) {
  test.describe(`in the ${scheme} half of the theme`, () => {
    test.use({ colorScheme: scheme })

    test('reads the text of the buttons of the decks, the Review button on the accent among them', async ({
      page,
    }) => {
      await openDecks(page)
      const review = page.getByRole('button', { name: /Review/ })
      await expect(review).toBeEnabled()
      await expectReadable(page.locator('body'), 'button', 2)
      await review.hover()
      await expectReadable(review, 'button', 1)
    })

    test('reads the text of the buttons of a card under review', async ({ page }) => {
      await openDecks(page)
      await page.getByRole('button', { name: /Review/ }).click()
      const show = page.getByRole('button', { name: /Show the answer/ })
      await expect(show).toBeVisible()
      await expectReadable(page.locator('body'), 'button', 2)
      await show.click()
      await expect(page.getByRole('button', { name: /Good/ })).toBeVisible()
      await expectReadable(page.locator('body'), 'button', 4)
    })
  })
}
