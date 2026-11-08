import { test, expect } from '../../support/test.js'
import { openVaultFile, openWorkspace, runCommand } from '../../support/workspace.js'
import { copyVaultFile, deleteVaultFiles } from '../../support/media.js'
import { createUniqueTitle } from '../../support/vault.js'

/** What the mock transcriber of `-debug-mock-ml` returns for every recording. */
const MOCK_TRANSCRIPT = 'Mock transcribed text for automated test.'

test.describe('the times of a transcript', () => {
  let name: string

  test.beforeEach(() => {
    name = `${createUniqueTitle('quay-talk').replace(/ /g, '-')}.wav`
    copyVaultFile('assets/lecture.wav', `assets/${name}`)
  })

  test.afterEach(() => {
    deleteVaultFiles(`assets/${name}`)
  })

  test('stand against each line, and a click on one seeks the player to it', async ({ page }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', name)
    await runCommand(page, 'Transcribe', /^Transcribe/)
    await expect(tab.getByText(MOCK_TRANSCRIPT)).toBeVisible({ timeout: 30_000 })
    const time = tab.locator('.cm-time', { hasText: /^0:00$/ })
    await expect(time).toHaveCount(1)

    const bar = tab.getByRole('slider', { name: 'Recording' })
    await tab.getByRole('button', { name: /^Play/ }).click()
    await expect(bar).toHaveAttribute('aria-valuetext', /^0:0[2-5]$/)
    await tab.getByRole('button', { name: /^Pause/ }).click()

    await time.click()

    await expect(bar).toHaveAttribute('aria-valuetext', '0:00')
  })
})
