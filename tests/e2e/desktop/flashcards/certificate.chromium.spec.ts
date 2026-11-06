/**
 * Chromium aborts the first handshake with a certificate it does not trust, and the flashcards
 * debug server logs a handshake error for it, even when the context ignores certificate errors. A
 * context that checks certificates opens the window only when the browser trusts the server's
 * certificate.
 */
import { test, expect } from '../tests/support/certificate.js'

test.describe('Flashcards debug server certificate in Chromium', () => {
  test('opens the window in a context that checks certificates', async ({ browser, baseURL }) => {
    const context = await browser.newContext({ baseURL, ignoreHTTPSErrors: false })
    try {
      const page = await context.newPage()
      await page.goto('/')
      await expect(page).toHaveTitle('numen flashcards')
    } finally {
      await context.close()
    }
  })
})
