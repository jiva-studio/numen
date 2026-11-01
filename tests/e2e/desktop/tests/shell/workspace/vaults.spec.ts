import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { test, expect, readSavedStates, type APIRequestContext, type Page } from '../../support/test.js'
import { getPaletteField, openWorkspace, runPaletteCommand } from '../../support/workspace.js'
import {
  addVault,
  callRPC,
  createUniqueTitle,
  getNoteTab,
  getRunVault,
  getVaultPath,
  listVaults,
  openNote,
  openVault,
  removeVault,
  type Vault,
} from '../../support/vault.js'

/** A folder of this machine outside the run's vault, holding one invented note, added as a vault. */
const addSecondVault = async (request: APIRequestContext): Promise<Vault> => {
  const folder = fs.mkdtempSync(path.join(os.tmpdir(), 'numen-e2e-second-vault-'))
  fs.writeFileSync(path.join(folder, 'Lighthouse.md'), '# Lighthouse\n\nThe lamp turns all night.\n')
  return addVault(request, folder, createUniqueTitle('Coastline'))
}

/** Puts the run's vault back in the window, and takes the second vault and its folder away. */
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

/** Picks a vault in the step a vault command opens on. */
const pickVault = async (page: Page, name: string) => {
  await getPaletteField(page).fill(name)
  await page.getByRole('option', { name: new RegExp(`^${name}`) }).click()
}

/** Where this machine's trash keeps a folder the core erased, by the folder's name. */
const getTrashedPath = async (request: APIRequestContext, name: string): Promise<string> =>
  process.platform === 'darwin'
    ? path.join(os.homedir(), '.Trash', name)
    : path.join(getVaultPath(), '.data', 'Trash', 'files', name)

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

  test('renames the vault in the window and leaves its folder as it is', async ({
    page,
    request,
  }) => {
    const before = await getRunVault(request)
    const name = createUniqueTitle('Harbour archive')
    try {
      await openWorkspace(page)

      await runPaletteCommand(page, 'Rename vault', 'Rename vault')
      await expect(getPaletteField(page)).toHaveValue(before.name)
      await getPaletteField(page).fill(name)
      await page.keyboard.press('Enter')

      await expect
        .poll(async () => (await listVaults(request)).find((one) => one.id === before.id)?.name)
        .toBe(name)
      expect((await getRunVault(request)).path).toBe(before.path)
      expect(fs.existsSync(path.join(before.path, 'Thermodynamics.md'))).toBe(true)
    } finally {
      await callRPC(request, 'VaultsService/RenameVault', { id: before.id, name: before.name })
    }
  })

  test('takes a vault off the list with Remove from list and keeps its folder', async ({
    page,
    request,
  }) => {
    const runVault = await getRunVault(request)
    const second = await addSecondVault(request)
    try {
      await openWorkspace(page)

      await runPaletteCommand(page, 'Remove from list', 'Remove from list')
      await pickVault(page, second.name)
      await page.getByRole('option', { name: new RegExp(`^Remove “${second.name}”`) }).click()

      await expect
        .poll(async () => (await listVaults(request)).map((one) => one.id))
        .not.toContain(second.id)
      expect(fs.readFileSync(path.join(second.path, 'Lighthouse.md'), 'utf8')).toContain(
        'The lamp turns all night.',
      )
    } finally {
      await dropSecondVault(request, second, runVault.id)
    }
  })

  test('erases a vault into the trash only once its name is typed back', async ({
    page,
    request,
  }) => {
    const runVault = await getRunVault(request)
    const second = await addSecondVault(request)
    const trashed = await getTrashedPath(request, path.basename(second.path))
    try {
      await openWorkspace(page)

      await runPaletteCommand(page, 'Erase vault', 'Erase vault')
      await pickVault(page, second.name)
      const erase = page.getByRole('option', { name: /^Erase “/ })
      await getPaletteField(page).fill(`${second.name} wrong`)
      await expect(erase).toHaveAttribute('aria-disabled', 'true')
      await page.keyboard.press('Enter')
      expect(fs.existsSync(path.join(second.path, 'Lighthouse.md'))).toBe(true)

      await getPaletteField(page).fill(second.name)
      await expect(erase).not.toHaveAttribute('aria-disabled', 'true')
      await page.keyboard.press('Enter')

      await expect
        .poll(async () => (await listVaults(request)).map((one) => one.id))
        .not.toContain(second.id)
      expect(fs.existsSync(second.path)).toBe(false)
      expect(fs.readFileSync(path.join(trashed, 'Lighthouse.md'), 'utf8')).toContain(
        'The lamp turns all night.',
      )
    } finally {
      fs.rmSync(trashed, { recursive: true, force: true })
      await dropSecondVault(request, second, runVault.id)
    }
  })
})
