import { test } from 'playwright/test'
import { openVaultFile, openWorkspace, waitForPassage } from '../support/workspace.js'

/** The words of the text layer the scanner wrote into the file. */
const LAYER_WORDS = 'shipping manifest'
const PATH = 'assets/flywheel.pdf'

test.describe('With the OCR plugin disabled', () => {
  test.describe.configure({ timeout: 180_000 })

  test('the text layer of the file is found', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'flywheel.pdf')
    await waitForPassage(page, LAYER_WORDS, PATH)
  })
})
