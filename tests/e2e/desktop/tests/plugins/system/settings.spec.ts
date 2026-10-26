import { test, expect, type Page } from 'playwright/test'
import { openWorkspace, runPaletteCommand } from '../../support/workspace.js'

/** The settings page of the transcription plugin, opened from the palette. The tab opens again on the plugin it last showed. */
async function openTranscriptionSettings(page: Page): Promise<void> {
  await runPaletteCommand(page, 'Settings', 'Settings')
  await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })
  const pluginRow = page.getByRole('button', { name: /Audio\/Video Transcription/i }).first()
  const model = page.getByTestId('setting-model')
  await expect(pluginRow.or(model)).toBeVisible({ timeout: 10_000 })
  if (await pluginRow.isVisible()) await pluginRow.click()
  await expect(page.getByTestId('setting-model')).toBeVisible({ timeout: 5_000 })
}

test.describe('Plugin Settings and Configuration', () => {
  test('opens command palette and navigates to Settings tab', async ({ page }) => {
    await openWorkspace(page)

    await runPaletteCommand(page, 'Settings', 'Settings')

    // Verify Settings tab opened
    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('Audio/Video Transcription').first()).toBeVisible({
      timeout: 10_000,
    })
  })

  test('shows the transcription model and the language it detects', async ({ page }) => {
    await openWorkspace(page)
    await openTranscriptionSettings(page)

    const model = page.getByTestId('setting-model')
    await expect(model.locator('.settings__choice')).toHaveText('Parakeet TDT 0.6B v3')
    await expect(model.locator('.settings__origin')).toHaveText('Default')
    await expect(page.getByText('25 European languages, 670 MB')).toBeVisible()

    const language = page.getByTestId('setting-language')
    await expect(language.locator('.settings__fixed')).toHaveText('Detected by the model')
    await expect(language.locator('.settings__origin')).toHaveText('Model default')
    await expect(language.locator('.settings__choice')).toHaveCount(0)
    await expect(page.getByText('Transcription Language')).toHaveCount(0)
  })

  test('changes the language with the model, and keeps the model across reopening', async ({
    page,
  }) => {
    await openWorkspace(page)
    await openTranscriptionSettings(page)

    const model = page.getByTestId('setting-model')
    await model.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: 'Parakeet TDT 0.6B v2', exact: true }).click()
    await expect(model.locator('.settings__choice')).toHaveText('Parakeet TDT 0.6B v2')
    await expect(model.locator('.settings__origin')).toHaveText('Chosen')

    const language = page.getByTestId('setting-language')
    await expect(language.locator('.settings__fixed')).toHaveText('English')
    await expect(language.locator('.settings__origin')).toHaveText('Model default')

    await page
      .getByRole('button', { name: /Close Settings/i })
      .first()
      .click()
    await expect(page.locator('.settings__plugins')).not.toBeVisible({ timeout: 5_000 })
    await openTranscriptionSettings(page)
    await expect(page.getByTestId('setting-model').locator('.settings__choice')).toHaveText(
      'Parakeet TDT 0.6B v2',
    )
    await expect(page.getByTestId('setting-language').locator('.settings__fixed')).toHaveText(
      'English',
    )

    await page.getByTestId('setting-model').locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: 'Parakeet TDT 0.6B v3', exact: true }).click()
    await expect(page.getByTestId('setting-language').locator('.settings__fixed')).toHaveText(
      'Detected by the model',
    )
  })

  test('searches palette and executes Files navigation command', async ({ page }) => {
    await openWorkspace(page)

    await runPaletteCommand(page, 'Files', 'Show files')

    // Verify file tree is open and assets folder is visible
    await expect(page.locator('[data-tree-row="assets"]').first()).toBeVisible({ timeout: 10_000 })
  })
})

test.describe('OCR settings', () => {
  test('offers the recognition models, and the custom model files only for the custom model', async ({
    page,
  }) => {
    await openWorkspace(page)
    await runPaletteCommand(page, 'Settings', 'Settings')
    const pluginRow = page.getByRole('button', { name: /Document OCR/i }).first()
    const model = page.getByTestId('setting-model')
    await expect(pluginRow.or(model)).toBeVisible({ timeout: 10_000 })
    if (await pluginRow.isVisible()) await pluginRow.click()

    await expect(model.locator('.settings__choice')).toHaveText('PP-OCRv6 tiny')
    await expect(page.getByTestId('setting-dpi').locator('.settings__choice')).toHaveText('300 dpi')
    await expect(page.getByText('Model file', { exact: true })).toHaveCount(0)

    await model.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: 'PP-OCRv5 East Slavic', exact: true }).click()
    await expect(page.getByText('Russian, Ukrainian, Belarusian, English, 7.9 MB')).toBeVisible()

    await model.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: 'Custom model', exact: true }).click()
    await expect(
      page.getByText('Languages unknown: they are those of the model file'),
    ).toBeVisible()
    await expect(page.getByText('Model file', { exact: true })).toBeVisible()
    await expect(page.getByText('Character dictionary', { exact: true })).toBeVisible()
    await expect(page.getByTestId('setting-path').first()).toHaveText('None')
    await expect(page.getByRole('button', { name: 'Choose…' })).toHaveCount(2)

    await model.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: 'PP-OCRv6 tiny', exact: true }).click()
    await expect(page.getByText('Model file', { exact: true })).toHaveCount(0)
  })
})

test.describe('Embeddings settings', () => {
  test('shows the settings of the chosen provider, and keeps the values of the others', async ({
    page,
  }) => {
    await openWorkspace(page)
    await runPaletteCommand(page, 'Settings', 'Settings')
    const pluginRow = page.getByRole('button', { name: /Vector Embeddings/i }).first()
    const provider = page.getByTestId('setting-provider')
    await expect(pluginRow.or(provider)).toBeVisible({ timeout: 10_000 })
    if (await pluginRow.isVisible()) await pluginRow.click()

    await expect(provider.locator('.settings__choice')).toHaveText('Local')
    await expect(
      page.getByText('BGE-M3 on this machine: 100+ languages, 1024 dimensions'),
    ).toBeVisible()
    await expect(page.getByText('Service URL', { exact: true })).toHaveCount(0)
    await expect(page.getByText('Model', { exact: true })).toHaveCount(0)
    await expect(page.getByText('API key', { exact: true })).toHaveCount(0)

    await provider.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: 'Custom service', exact: true }).click()
    await expect(page.getByText('Service URL', { exact: true })).toBeVisible()
    await expect(page.getByText('Model', { exact: true })).toBeVisible()
    await expect(page.getByText('API key', { exact: true }).first()).toBeVisible()
    const url = page.getByRole('textbox', { name: 'Service URL' })
    await url.fill('http://127.0.0.1:9/v1')
    await url.press('Tab')
    await expect(page.getByRole('textbox', { name: 'Service URL' })).not.toBeFocused()

    await provider.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: 'OpenAI', exact: true }).click()
    await expect(page.getByText('Service URL', { exact: true })).toHaveCount(0)
    await expect(page.getByText('Model', { exact: true })).toBeVisible()

    await provider.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: 'Custom service', exact: true }).click()
    await expect(page.getByRole('textbox', { name: 'Service URL' })).toHaveValue(
      'http://127.0.0.1:9/v1',
    )

    await provider.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: 'Local', exact: true }).click()
    await expect(page.getByText('Service URL', { exact: true })).toHaveCount(0)
  })
})
