import { expect, test } from '../../support/test.js'
import { openWorkspace, runCommand } from '../../support/workspace.js'

test('Starting audio recording in a window without a recorder explains why it stays idle', async ({
  page,
}) => {
  await openWorkspace(page)
  await runCommand(page, 'Record audio', /^Record audio$/i)
  const recorder = page.getByRole('region', { name: 'Audio Recorder' })
  await recorder.getByRole('button', { name: 'Start recording' }).click()
  await expect(recorder).toContainText('Microphone recording is unavailable in this window')
  await expect(recorder.getByRole('button', { name: 'Start recording' })).toBeEnabled()
  await expect(recorder.getByRole('button', { name: 'Stop recording' })).toHaveCount(0)
})
