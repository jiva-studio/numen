import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace, runPaletteCommand } from '../../support/workspace.js'
import { writeSetting } from '../../support/vault.js'

async function openTranscriptionSettings(page: Page): Promise<void> {
  await runPaletteCommand(page, 'Open settings', 'Open settings')
  await page
    .getByRole('button', { name: /Audio\/Video Transcription/i })
    .first()
    .click()
  await expect(page.getByTestId('setting-model')).toBeVisible({ timeout: 5_000 })
}

async function reopenSettings(page: Page): Promise<void> {
  await runPaletteCommand(page, 'Open settings', 'Open settings')
  await expect(page.getByTestId('setting-model')).toBeVisible({ timeout: 5_000 })
}

test.describe('Plugin Settings and Configuration', () => {
  test('opens command palette and navigates to Settings tab', async ({ page }) => {
    await openWorkspace(page)

    await runPaletteCommand(page, 'Open settings', 'Open settings')

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

  test('changes the transcription model and keeps the model across reopening', async ({ page }) => {
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
    await expect(page.locator('.settings__plugins')).toBeHidden({ timeout: 5_000 })
    await reopenSettings(page)
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

    await runPaletteCommand(page, 'file tree', 'Open file tree')

    // Verify file tree is open and assets folder is visible
    await expect(page.locator('[data-tree-row="assets"]').first()).toBeVisible({ timeout: 10_000 })
  })

  test('keeps cleared plugin search query when navigating into a plugin detail and back', async ({
    page,
  }) => {
    await openWorkspace(page)
    await runPaletteCommand(page, 'Open settings', 'Open settings')
    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })

    const searchInput = page.getByRole('searchbox', { name: 'Search plugins' }).first()
    await expect(searchInput).toBeVisible({ timeout: 5_000 })
    await searchInput.fill('feature:agent')
    await expect(searchInput).toHaveValue('feature:agent')

    await searchInput.fill('')
    await expect(searchInput).toHaveValue('')

    const pluginRow = page
      .locator('.settings__plugins')
      .getByRole('button', { name: /Agent Chat/i })
      .first()
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
    await runPaletteCommand(page, 'Open settings', 'Open settings')
    await page
      .getByRole('button', { name: /Document OCR/i })
      .first()
      .click()
    const model = page.getByTestId('setting-model')
    await expect(model).toBeVisible({ timeout: 10_000 })

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
    await expect(page.getByTestId('setting-path')).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Choose…' })).toHaveCount(2)

    await model.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: /^PP-OCRv6 tiny/ }).click()
    await expect(page.getByText('Model file', { exact: true })).toHaveCount(0)
  })
})

test.describe('Embeddings settings', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, ['md.numen.engines.embeddings', 'source'], 'local')
  })

  test('shows the model for the local source and the connection for the other source', async ({
    page,
  }) => {
    await openWorkspace(page)
    await runPaletteCommand(page, 'Open settings', 'Open settings')
    await page
      .getByRole('button', { name: /Vector Embeddings/i })
      .first()
      .click()
    const source = page.getByTestId('setting-source')
    await expect(source).toBeVisible({ timeout: 10_000 })

    await expect(source.locator('.settings__choice')).toHaveText('Local model')
    await expect(page.getByText('Model', { exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Add connection' })).toHaveCount(0)
    await expect(page.getByText('Service URL', { exact: true })).toHaveCount(0)
    await expect(page.getByText('API key', { exact: true })).toHaveCount(0)

    await source.locator('.settings__choice').click()
    await expect(
      page.getByRole('menuitemradio', {
        name: /Local model.*Runs an embedding model on this machine/i,
      }),
    ).toBeVisible()
    await page.getByRole('menuitemradio', { name: /^Connection/ }).click()
    await expect(page.getByRole('button', { name: 'Add connection' })).toBeVisible()
    await expect(page.getByText('Model', { exact: true })).toHaveCount(0)

    await source.locator('.settings__choice').click()
    await page.getByRole('menuitemradio', { name: /^Local model/ }).click()
    await expect(page.getByRole('button', { name: 'Add connection' })).toHaveCount(0)
    await expect(page.getByText('Model', { exact: true })).toBeVisible()
  })
})
