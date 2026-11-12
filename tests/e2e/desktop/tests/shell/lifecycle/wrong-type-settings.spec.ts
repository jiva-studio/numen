import { test, expect } from '../../support/test.js'
import { startOwnServer, type OwnServer } from '../../support/server.js'
import { expectWorkspaceDrawn, USABLE_WITHIN_MS } from '../../support/workspace.js'

/** A copy of an old settings file: the keys of the old layout, and objects under keys that are no plugin ids. */
const SETTINGS = JSON.stringify({
  v: 1,
  importing: {
    yt_dlp: { environment: { SSL_CERT_FILE: '/etc/ssl/certs/ca-certificates.crt' } },
    ffmpeg: { command: ['ffmpeg', '-y'] },
  },
  review: { day_start_time: '05:00' },
  plugins: { 'md.numen.tools.media.importer': { ytDlp: { environment: { A: 'b' } } } },
  'md.numen.proofreading': { enabled: true },
})

test.describe('Start-up with old keys and values of the wrong type in the settings file', () => {
  let server: OwnServer

  test.beforeAll(async () => {
    server = await startOwnServer([], { settings: SETTINGS })
  })

  test.afterAll(async () => {
    await server?.stop()
  })

  test('opens the window and shows notices that name the old keys, none of them a failed task', async ({
    page,
  }) => {
    await page.goto(server.url)
    await expectWorkspaceDrawn(page)
    await expect(page.getByText(/Numen can't start/)).toHaveCount(0)

    await page.locator('[data-tasks-status] button').click()
    const popover = page.locator('.task-popover')
    await expect(popover).toBeVisible({ timeout: USABLE_WITHIN_MS })
    for (const key of ['importing', 'review', 'plugins', 'md.numen.proofreading']) {
      await expect(popover.locator('.task-item').filter({ hasText: key })).toHaveCount(1, {
        timeout: USABLE_WITHIN_MS,
      })
    }
    await expect(
      popover.locator('.task-item--failed').filter({ hasText: 'Read settings' }),
    ).toHaveCount(0)
  })
})
