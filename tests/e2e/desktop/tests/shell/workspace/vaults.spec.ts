import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import {
  test,
  expect,
  readSavedStates,
  type APIRequestContext,
  type Page,
} from '../../support/test.js'
import { getPaletteField, openWorkspace, runPaletteCommand } from '../../support/workspace.js'
import {
  addVault,
  createUniqueTitle,
  getNoteTab,
  getRunVault,
  listVaults,
  openNote,
  openVault,
  removeVault,
  type Vault,
} from '../../support/vault.js'

const addSecondVault = async (request: APIRequestContext): Promise<Vault> => {
  const folder = fs.mkdtempSync(path.join(os.tmpdir(), 'numen-e2e-second-vault-'))
  fs.writeFileSync(
    path.join(folder, 'Lighthouse.md'),
    '# Lighthouse\n\nThe lamp turns all night.\n',
  )
  return addVault(request, folder, createUniqueTitle('Coastline'))
}

const dropSecondVault = async (
  request: APIRequestContext,
  second: Vault,
  runId: string,
): Promise<void> => {
  await openVault(request, runId)
  if ((await listVaults(request)).some((one) => one.id === second.id)) {
    await removeVault(request, second.id)
  }
  fs.rmSync(second.path, { recursive: true, force: true })
}

const pickVault = async (page: Page, name: string) => {
  await getPaletteField(page).fill(name)
  await page.getByRole('option', { name: new RegExp(`^${name}`) }).click()
}

test.describe('Vaults of the installation', () => {
  test('switches the window to another vault with Open vault', async ({ page, request }) => {
    const runVault = await getRunVault(request)
    const second = await addSecondVault(request)
    try {
      await openWorkspace(page)
      await openNote(page, 'notes/Entropy.md')

      await runPaletteCommand(page, 'Open vault', 'Open vault')
      await pickVault(page, second.name)

      await expect(page.getByRole('tab', { name: 'Lighthouse Close Lighthouse' })).toBeVisible()
      await page.getByRole('tab', { name: /^Files/ }).click()
      await expect(page.locator('[data-tree-row="Lighthouse.md"]')).toBeVisible()
      await expect(page.locator('[data-tree-row="notes"]')).toHaveCount(0)
      await expect(getNoteTab(page, 'notes/Entropy.md')).toHaveCount(0)
      expect([...(await readSavedStates(request)).keys()]).not.toContain('notes/Entropy.md')
    } finally {
      await dropSecondVault(request, second, runVault.id)
    }
  })
})
