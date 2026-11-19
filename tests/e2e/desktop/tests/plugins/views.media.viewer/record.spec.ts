import { expect, test } from '../../support/test.js'
import { openWorkspace, runCommand } from '../../support/workspace.js'

test('Stop saves the captured audio and opens a playable vault recording', async ({ page }) => {
  await openWorkspace(page)
  await runCommand(page, 'Record audio', /^Record audio$/i)
  await page.evaluate(() => {
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: {
        async getUserMedia() {
          const context = new AudioContext()
          const source = context.createOscillator()
          const output = context.createMediaStreamDestination()
          source.connect(output)
          source.start()
          await context.resume()
          return output.stream
        },
      },
    })
  })
  const recorder = page.getByRole('region', { name: 'Audio Recorder' })
  await recorder.getByRole('button', { name: 'Start recording' }).click()
  await expect(recorder).toContainText(/0:0[1-9]/)
  await recorder.getByRole('button', { name: 'Stop recording' }).click()
  const player = page.getByRole('group', { name: 'Recording' })
  await expect(player).toBeVisible()
  await expect
    .poll(async () => Number(await player.getByRole('slider').getAttribute('max')))
    .toBeGreaterThan(0)
})
