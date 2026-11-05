import { test, expect, type Page } from '../../support/test.js'
import { openVaultFile, openWorkspace, runCommand } from '../../support/workspace.js'
import { getRow, readWidth, showInFiles } from '../../support/layout.js'
import { callRPC, writeSetting } from '../../support/vault.js'

const NOTES_ID = 'md.numen.notes'
const PDF_ID = 'md.numen.readers.pdf'
const EMBEDDINGS_ID = 'md.numen.engines.embeddings'

/** The names of the secrets that have a key stored for the plugin. The key itself is never answered. */
const readStoredSecrets = async (page: Page, pluginId: string): Promise<string[]> => {
  const answer = await callRPC(page.request, 'SettingsService/GetSettings', {})
  const plugins = (answer['plugins'] ?? []) as { id?: string; storedSecrets?: string[] }[]
  return plugins.find((one) => one.id === pluginId)?.storedSecrets ?? []
}

test.describe('Plugin settings take effect', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, [NOTES_ID, 'line_numbers'], false)
    await writeSetting(request, [NOTES_ID, 'soft_wrap'], true)
    await writeSetting(request, [PDF_ID, 'default_zoom'], 'page')
    await writeSetting(request, [EMBEDDINGS_ID, 'provider'], 'local')
    await callRPC(request, 'SettingsService/DeletePluginSecret', {
      pluginId: EMBEDDINGS_ID,
      name: 'apiKey',
    }).catch(() => undefined)
  })

  test('draws line numbers and no soft wrap in a note once the notes settings say so', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await showInFiles(page, ['notes'])
    await getRow(page, 'notes/Entropy.md').press('Enter')
    const editor = page.getByRole('tabpanel', { name: /Entropy/ }).locator('.cm-editor')
    await expect(editor).toBeVisible()
    await expect(editor.locator('.cm-lineNumbers')).toHaveCount(0)
    await expect(editor.locator('.cm-lineWrapping')).toHaveCount(1)

    await writeSetting(request, [NOTES_ID, 'line_numbers'], true)
    await writeSetting(request, [NOTES_ID, 'soft_wrap'], false)

    await expect(editor.locator('.cm-lineNumbers')).toHaveCount(1)
    await expect(editor.locator('.cm-lineWrapping')).toHaveCount(0)
  })

  for (const [fit, getExpected] of [
    ['width', (viewport: number) => viewport - 32],
    ['100%', () => (612 * 96) / 72],
  ] as const) {
    test(`draws a new PDF at the ${fit} default zoom`, async ({ page, request }) => {
      await writeSetting(request, [PDF_ID, 'default_zoom'], fit)
      await openWorkspace(page)
      await openVaultFile(page, 'assets', 'pages.pdf')
      const first = page.locator('.reader__page[data-page="0"]')
      await expect(first).toBeAttached({ timeout: 20_000 })
      const viewport = page.locator('.reader__viewport')

      await expect
        .poll(async () =>
          Math.abs((await readWidth(first)) - getExpected(await readWidth(viewport))),
        )
        .toBeLessThan(24)
    })
  }

  test('keeps an API key out of the page, shows it as set, and clears it with the delete button', async ({
    page,
    request,
  }) => {
    await writeSetting(request, [EMBEDDINGS_ID, 'provider'], 'custom')
    await openWorkspace(page)
    await runCommand(page, 'Open settings', /^Open settings$/)
    const settings = page.getByRole('tabpanel', { name: /Settings/ })
    await settings.getByRole('button', { name: /Vector Embeddings/ }).click()
    const field = settings.getByRole('textbox', { name: 'API key' })
    await expect(field).toHaveAttribute('type', 'password')
    await expect(settings.getByRole('button', { name: 'Delete the key' })).toHaveCount(0)
    expect(await readStoredSecrets(page, EMBEDDINGS_ID)).toEqual([])

    await field.fill('sk-invented-harbour-key')
    await field.press('Enter')

    await expect.poll(() => readStoredSecrets(page, EMBEDDINGS_ID)).toEqual(['apiKey'])
    await expect(field).toHaveValue('')
    await expect(field).toHaveAttribute('placeholder', '••••••••')

    await page.reload()
    await runCommand(page, 'Open settings', /^Open settings$/)
    const reloaded = page.getByRole('tabpanel', { name: /Settings/ })
    await reloaded.getByRole('button', { name: /Vector Embeddings/ }).click()
    await expect(reloaded.getByRole('textbox', { name: 'API key' })).toHaveValue('')
    await expect(reloaded.getByRole('textbox', { name: 'API key' })).toHaveAttribute(
      'placeholder',
      '••••••••',
    )
    expect(await page.content()).not.toContain('sk-invented-harbour-key')

    const deleteButton = reloaded.getByRole('button', { name: 'Delete the key' })
    await expect(deleteButton).toHaveAttribute('title', 'Delete the key')
    await deleteButton.click()

    await expect.poll(() => readStoredSecrets(page, EMBEDDINGS_ID)).toEqual([])
    await expect(reloaded.getByRole('button', { name: 'Delete the key' })).toHaveCount(0)
  })
})
