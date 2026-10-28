import { test, expect, type Page } from '../../support/test.js'
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

  test('shows the transcription model and its supported languages', async ({ page }) => {
    await openWorkspace(page)
    await openTranscriptionSettings(page)

    const model = page.getByTestId('setting-model')
    await expect(model.locator('.settings__choice')).toHaveText('Parakeet TDT 0.6B v3')

    await model.locator('.settings__choice').click()
    await expect(
      page.getByRole('menuitemradio', { name: /Parakeet TDT 0\.6B v3.*25 European languages/i }),
    ).toBeVisible()
    await page.keyboard.press('Escape')
  })

  test('changes the transcription model and keeps the model across reopening', async ({
    page,
  }) => {
    await openWorkspace(page)
    await openTranscriptionSettings(page)

    const model = page.getByTestId('setting-model')
    await model.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: /^Parakeet TDT 0\.6B v2/ }).click()
    await expect(model.locator('.settings__choice')).toHaveText('Parakeet TDT 0.6B v2')

    await page
      .getByRole('button', { name: /Close Settings/i })
      .first()
      .click()
    await expect(page.locator('.settings__plugins')).not.toBeVisible({ timeout: 5_000 })
    await openTranscriptionSettings(page)
    await expect(page.getByTestId('setting-model').locator('.settings__choice')).toHaveText(
      'Parakeet TDT 0.6B v2',
    )

    await page.getByTestId('setting-model').locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: /^Parakeet TDT 0\.6B v3/ }).click()
    await expect(page.getByTestId('setting-model').locator('.settings__choice')).toHaveText(
      'Parakeet TDT 0.6B v3',
    )
  })

  test('searches palette and executes Files navigation command', async ({ page }) => {
    await openWorkspace(page)

    await runPaletteCommand(page, 'Files', 'Open files tab')

    // Verify file tree is open and assets folder is visible
    await expect(page.locator('[data-tree-row="assets"]').first()).toBeVisible({ timeout: 10_000 })
  })

  test('keeps cleared plugin search query when navigating into a plugin detail and back', async ({
    page,
  }) => {
    await openWorkspace(page)
    await runPaletteCommand(page, 'Settings', 'Settings')
    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })

    const searchInput = page.getByRole('searchbox', { name: 'Search plugins' }).first()
    await expect(searchInput).toBeVisible({ timeout: 5_000 })
    await searchInput.fill('feature:agent')
    await expect(searchInput).toHaveValue('feature:agent')

    await searchInput.fill('')
    await expect(searchInput).toHaveValue('')

    const pluginRow = page.getByRole('button', { name: /Agent Chat/i }).first()
    await expect(pluginRow).toBeVisible({ timeout: 5_000 })
    await pluginRow.click()

    const backBtn = page.getByRole('button', { name: /All plugins/i }).first()
    await expect(backBtn).toBeVisible({ timeout: 5_000 })
    await backBtn.click()

    await expect(page.getByRole('searchbox', { name: 'Search plugins' }).first()).toHaveValue('')
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
    await expect(
      page.getByRole('menuitemradio', {
        name: /PP-OCRv5 East Slavic.*Russian, Ukrainian, Belarusian, English, 7\.9 MB/i,
      }),
    ).toBeVisible()
    await page.getByRole('menuitemradio', { name: /^PP-OCRv5 East Slavic/ }).click()
    await expect(model.locator('.settings__choice')).toHaveText('PP-OCRv5 East Slavic')

    await model.locator('.settings__choice').click()
    await expect(
      page.getByRole('menuitemradio', {
        name: /Custom model.*Languages unknown: they are those of the model file/i,
      }),
    ).toBeVisible()
    await page.getByRole('menuitemradio', { name: /^Custom model/ }).click()
    await expect(page.getByText('Model file', { exact: true })).toBeVisible()
    await expect(page.getByText('Character dictionary', { exact: true })).toBeVisible()
    await expect(page.getByTestId('setting-path').first()).toHaveText('None')
    await expect(page.getByRole('button', { name: 'Choose…' })).toHaveCount(2)

    await model.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: /^PP-OCRv6 tiny/ }).click()
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
    await expect(page.getByText('Service URL', { exact: true })).toHaveCount(0)
    await expect(page.getByText('Model', { exact: true })).toBeVisible()
    await expect(page.getByText('API key', { exact: true })).toHaveCount(0)

    await provider.locator('.settings__choice').click()
    await expect(
      page.getByRole('menuitemradio', {
        name: /Local.*BGE-M3 on this machine: 100\+ languages, 1024 dimensions/i,
      }),
    ).toBeVisible()
    await page.getByRole('menuitemradio', { name: /^Custom service/ }).click()
    await expect(page.getByText('Service URL', { exact: true })).toBeVisible()
    await expect(page.getByText('Model', { exact: true })).toBeVisible()
    await expect(page.getByText('API key', { exact: true }).first()).toBeVisible()
    const url = page.getByRole('textbox', { name: 'Service URL' })
    await url.fill('http://127.0.0.1:9/v1')
    await url.press('Tab')
    await expect(page.getByRole('textbox', { name: 'Service URL' })).not.toBeFocused()

    await provider.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: /^OpenAI/ }).click()
    await expect(page.getByText('Service URL', { exact: true })).toHaveCount(0)
    await expect(page.getByText('Model', { exact: true })).toBeVisible()

    await provider.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: /^Custom service/ }).click()
    await expect(page.getByRole('textbox', { name: 'Service URL' })).toHaveValue(
      'http://127.0.0.1:9/v1',
    )

    await provider.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: /^Local/ }).click()
    await expect(page.getByText('Service URL', { exact: true })).toHaveCount(0)
  })
})
