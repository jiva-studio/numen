import { test, expect } from '../../support/test.js'
import { startOwnServer, type OwnServer } from '../../support/server.js'
import { expectWorkspaceDrawn, USABLE_WITHIN_MS } from '../../support/workspace.js'

/** A copy of a settings file whose old entry of the importer holds an object where a text is declared. */
const SETTINGS = JSON.stringify({
  v: 1,
  importing: { yt_dlp: { environment: { SSL_CERT_FILE: '/etc/ssl/certs/ca-certificates.crt' } } },
})

test.describe('Start-up with a value of the wrong type in the settings file', () => {
  let server: OwnServer

  test.beforeAll(async () => {
    server = await startOwnServer([], { settings: SETTINGS })
  })

  test.afterAll(async () => {
    await server.stop()
  })

  test('opens the window and shows a notice that names the setting', async ({ page }) => {
    await page.goto(server.url)
    await expectWorkspaceDrawn(page)
    await expect(page.getByText(/Numen can't start/)).toHaveCount(0)

    await page.locator('.status-bar__trigger').click()
    const popover = page.locator('.task-popover')
    await expect(popover).toBeVisible({ timeout: USABLE_WITHIN_MS })
    await expect(popover).toContainText(/md\.numen\.tools\.media\.importer\.ytDlp/, {
      timeout: USABLE_WITHIN_MS,
    })
  })
})
