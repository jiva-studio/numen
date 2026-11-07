import { test, expect, type Page } from '../../support/test.js'
import { openVaultFile, openWorkspace, runCommand } from '../../support/workspace.js'
import { getRow, readWidth, showInFiles } from '../../support/layout.js'
import { callRPC, writeSetting } from '../../support/vault.js'

const NOTES_ID = 'md.numen.notes'
const PDF_ID = 'md.numen.readers.pdf'
const EMBEDDINGS_ID = 'md.numen.engines.embeddings'

interface ListedSecret {
  readonly id: string
  readonly name: string
  readonly group: string
  readonly hasValue: boolean
}

/** The secrets the core lists. A value is never answered. */
const readSecrets = async (page: Page): Promise<ListedSecret[]> => {
  const answer = await callRPC(page.request, 'SettingsService/GetSettings', {})
  return (answer['secrets'] ?? []) as ListedSecret[]
}

/** Deletes every secret, so the next test starts with none. */
const deleteSecrets = async (page: Page): Promise<void> => {
  for (const one of await readSecrets(page)) {
    await callRPC(page.request, 'SettingsService/DeleteSecret', { id: one.id })
  }
}

test.describe('Plugin settings take effect', () => {
  test.afterEach(async ({ page, request }) => {
    await writeSetting(request, [NOTES_ID, 'lineNumbers'], false)
    await writeSetting(request, [NOTES_ID, 'softWrap'], true)
    await writeSetting(request, [PDF_ID, 'defaultZoom'], 'page')
    await writeSetting(request, [EMBEDDINGS_ID, 'provider'], 'local')
    await deleteSecrets(page)
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

    await writeSetting(request, [NOTES_ID, 'lineNumbers'], true)
    await writeSetting(request, [NOTES_ID, 'softWrap'], false)

    await expect(editor.locator('.cm-lineNumbers')).toHaveCount(1)
    await expect(editor.locator('.cm-lineWrapping')).toHaveCount(0)
  })

  for (const [fit, getExpected] of [
    ['width', (viewport: number) => viewport - 32],
    ['100%', () => (612 * 96) / 72],
  ] as const) {
    test(`draws a new PDF at the ${fit} default zoom`, async ({ page, request }) => {
      await writeSetting(request, [PDF_ID, 'defaultZoom'], fit)
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

  test('keeps a secret value out of the page after it is entered and deletes the secret after a confirmation', async ({
    page,
  }) => {
    const value = 'sk-invented-harbour-key'
    await writeSetting(page.request, [EMBEDDINGS_ID, 'provider'], 'openai')
    await openWorkspace(page)
    await runCommand(page, 'Open settings', /^Open settings$/)
    const settings = page.getByRole('tabpanel', { name: /Settings/ })

    await settings.getByRole('button', { name: /Vector Embeddings/ }).click()
    const field = settings.getByRole('textbox', { name: 'API key' })
    await expect(field).toHaveAttribute('type', 'password')
    await expect(field).not.toBeEditable()
    await settings.getByRole('button', { name: 'All plugins' }).click()

    await settings.getByRole('button', { name: /^Secrets$/ }).click()
    const section = settings.getByRole('region', { name: 'openai' })
    await section.getByLabel('Default').fill(value)
    await section.getByLabel('Default').press('Enter')

    await expect
      .poll(async () => (await readSecrets(page)).map((one) => one.name))
      .toEqual(['Default'])
    expect((await readSecrets(page))[0]).toMatchObject({ group: 'openai', hasValue: true })
    await expect(section.getByLabel('Default')).toHaveValue('')
    expect(await page.content()).not.toContain(value)

    await page.reload()
    await runCommand(page, 'Open settings', /^Open settings$/)
    const reloaded = page.getByRole('tabpanel', { name: /Settings/ })
    await reloaded.getByRole('button', { name: /^Secrets$/ }).click()
    expect(await page.content()).not.toContain(value)

    await reloaded
      .getByRole('listitem')
      .filter({ hasText: 'Default' })
      .getByRole('button', { name: /delete/i })
      .click()
    const confirmation = page.getByRole('alertdialog')
    await expect(confirmation).toBeVisible()
    await confirmation.getByRole('button', { name: /delete/i }).click()

    await expect.poll(() => readSecrets(page)).toEqual([])
  })
})
