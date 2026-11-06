import { test, expect } from '../support/test.js'
import {
  getPaletteField,
  openVaultFile,
  openWorkspace,
  runCommand,
  runPaletteCommand,
  waitForPassage,
} from '../support/workspace.js'
import { readOwner, writeSetting } from '../support/vault.js'

/** Words of the text the mock recogniser of `-debug-mock-ml` reads on every page. */
const MOCK_RECOGNISED_WORDS = 'recovers kinetic energy'

test.describe('A scanned document', () => {
  test.describe.configure({ timeout: 180_000 })

  test('is recognised, found through the search, and opened with the place highlighted', async ({
    page,
  }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'regenerative-braking.pdf')
    await expect(tab.getByRole('img', { name: 'Page 1' })).toBeVisible()

    await runCommand(page, 'Recognise', /^Recognise text$/)

    const passage = await waitForPassage(
      page,
      MOCK_RECOGNISED_WORDS,
      'assets/regenerative-braking.pdf',
    )
    await expect(passage).toContainText(/kinetic energy/i)
    await passage.click()

    const reader = page.getByRole('tabpanel', { name: /regenerative-braking/i })
    await expect(reader.getByRole('img', { name: 'Page 1' })).toBeVisible()
    await expect(reader.locator('.reader__highlight').first()).toBeVisible()
  })

  test('is offered the recognise command of the OCR plugin once', async ({ page }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'regenerative-braking.pdf')

    await page.keyboard.press('Control+p')
    await getPaletteField(page).fill('Recognise')

    await expect(page.getByRole('option', { name: /^Recognise text$/ })).toHaveCount(1)
  })
})

const OCR_ID = 'md.numen.engines.ocr'

/** The models the Model setting of the OCR plugin offers, in order, with what each reads. */
const OCR_MODELS: readonly { readonly title: string; readonly detail: string }[] = [
  {
    title: 'Multilingual',
    detail:
      'Latin-script languages with their diacritics, Cyrillic-script languages, Greek, Chinese, English, 28 MB',
  },
  { title: 'PP-OCRv6 tiny', detail: 'Chinese, English and Latin-script languages, 4.5 MB' },
  {
    title: 'PP-OCRv6 small',
    detail: 'Chinese, English, Japanese and Latin-script languages, 21 MB',
  },
  {
    title: 'PP-OCRv5 Latin',
    detail:
      'English, French, German, Spanish, Italian, Portuguese, Dutch, Polish, Czech, Slovak, Hungarian, Romanian, Croatian, Slovenian, Danish, Swedish, Norwegian, Finnish, Estonian, Lithuanian, Latvian, Turkish, Latin and other Latin-script languages, 7.9 MB',
  },
  { title: 'PP-OCRv5 East Slavic', detail: 'Russian, Ukrainian, Belarusian, English, 7.9 MB' },
  {
    title: 'PP-OCRv5 Cyrillic',
    detail:
      'Russian, Ukrainian, Belarusian, Bulgarian, Serbian, Macedonian, Mongolian, Kazakh, Kyrgyz, Tajik, Tatar and other Cyrillic-script languages, English, 8.1 MB',
  },
  {
    title: 'PP-OCRv5 Devanagari',
    detail: 'Hindi, Sanskrit, Marathi, Nepali, Konkani, Maithili, Bhojpuri, English, 7.9 MB',
  },
  { title: 'PP-OCRv5 Greek', detail: 'Greek, English, 7.8 MB' },
  { title: 'Custom model', detail: 'Languages unknown: they are those of the model file' },
]

test.describe('The model of the OCR plugin', () => {
  test.describe.configure({ timeout: 180_000 })

  test.afterEach(async ({ request }) => {
    await writeSetting(request, [OCR_ID, 'model'], 'multilingual')
  })

  test('is chosen in the palette from the models with their languages, and recognises with the one chosen', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    const tab = await openVaultFile(page, 'assets', 'regenerative-braking.pdf')
    await expect(tab.getByRole('img', { name: 'Page 1' })).toBeVisible()

    await runPaletteCommand(page, 'Document OCR: Model', 'Document OCR: Model')
    const rows = page.getByRole('option')
    await expect(rows.locator('[data-palette="name"]')).toHaveText(
      OCR_MODELS.map((one) => one.title),
    )
    await expect(rows.locator('[data-palette="detail"]')).toHaveText(
      OCR_MODELS.map((one) => one.detail),
    )
    await expect(rows.first()).toHaveAttribute('aria-selected', 'true')

    await rows
      .filter({ has: page.locator('[data-palette="name"]', { hasText: /^PP-OCRv5 Cyrillic$/ }) })
      .click()
    await expect(page.getByRole('dialog', { name: 'Search the vault' })).toHaveCount(0)
    await expect
      .poll(async () => await readOwner(request, OCR_ID))
      .toMatchObject({ model: 'pp-ocrv5-cyrillic' })

    await runCommand(page, 'Recognise', /^Recognise text$/)
    const passage = await waitForPassage(
      page,
      MOCK_RECOGNISED_WORDS,
      'assets/regenerative-braking.pdf',
    )
    await expect(passage).toContainText(/kinetic energy/i)
  })
})
