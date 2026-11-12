/**
 * The vault the debug server serves, read and written through its RPCs and, for what an RPC does
 * not answer (frontmatter, folders outside the vault), through the file system of this machine:
 * the server and the tests run on one machine.
 */
import fs from 'node:fs'
import path from 'node:path'
import { expect, type APIRequestContext, type Locator, type Page } from 'playwright/test'
import type { JsonObject, JsonValue } from './json.js'

import os from 'node:os'

/** The folder of the run's vault, which the server script wrote under the port it listens on. */
export const getVaultPath = (): string =>
  fs
    .readFileSync(
      path.join(os.tmpdir(), `numen-e2e-vault-path-${process.env.NUMEN_TEST_PORT ?? '3456'}`),
      'utf8',
    )
    .trim()

/** Calls a unary RPC of the debug server with a JSON body, and gives back its JSON answer. */
export const callRPC = async (
  request: APIRequestContext,
  method: string,
  data: object,
): Promise<JsonObject> => {
  const answer = await request.post(`/numen.v1.${method}`, {
    data,
    headers: { 'Content-Type': 'application/json' },
  })
  const text = await answer.text()
  if (!answer.ok()) throw new Error(`${method} answered ${answer.status()}: ${text}`)
  return text ? (JSON.parse(text) as JsonObject) : {}
}

/** One link a note is made with, as CreateNote takes it. */
interface NewLink {
  readonly to: string
  readonly role: 'ROLE_PARENT' | 'ROLE_CHILD' | 'ROLE_JUMP' | 'ROLE_REF'
}

/** Creates a note in `folder` and gives back its path. */
export const createNote = async (
  request: APIRequestContext,
  title: string,
  folder = 'notes',
  links: readonly NewLink[] = [],
): Promise<string> => {
  const answer = await callRPC(request, 'NoteService/CreateNote', { title, path: folder, links })
  if (answer['error'] || !answer['path']) {
    throw new Error(`CreateNote ${title} refused: ${JSON.stringify(answer)}`)
  }
  return String(answer['path'])
}

/** The prose of a note below its frontmatter, and the code of why it could not be read. */
export const readNote = async (
  request: APIRequestContext,
  notePath: string,
): Promise<{ body: string; error: string }> => {
  const answer = await callRPC(request, 'NoteService/ReadNote', { path: notePath })
  return { body: String(answer['body'] ?? ''), error: String(answer['error'] ?? '') }
}

/** Writes prose below the frontmatter of a note, as a program other than the window would. */
export const writeNote = async (
  request: APIRequestContext,
  notePath: string,
  body: string,
): Promise<void> => {
  const answer = await callRPC(request, 'NoteService/WriteNote', { path: notePath, body })
  if (answer['error']) throw new Error(`WriteNote ${notePath} refused: ${JSON.stringify(answer)}`)
}

/** Waits until the prose of the note on disk holds `text`. */
export const waitForSavedBody = async (
  request: APIRequestContext,
  notePath: string,
  text: string,
  timeout = 15_000,
): Promise<void> => {
  await expect
    .poll(async () => (await readNote(request, notePath)).body, { timeout })
    .toContain(text)
}

/** One entry of a folder listing. */
interface Entry {
  readonly path: string
  readonly name: string
  readonly isFolder: boolean
}

export const listFiles = async (request: APIRequestContext, folder = ''): Promise<Entry[]> => {
  const answer = await callRPC(request, 'FileService/ListFiles', { path: folder })
  return ((answer['entries'] as Partial<Entry>[] | undefined) ?? []).map((one) => ({
    path: one.path ?? '',
    name: one.name ?? '',
    isFolder: one.isFolder ?? false,
  }))
}

/** The paths of the files in one folder of the vault, folders left out. */
export const listFilePaths = async (request: APIRequestContext, folder = ''): Promise<string[]> =>
  (await listFiles(request, folder)).filter((one) => !one.isFolder).map((one) => one.path)

/**
 * Removes a note file and the copy it leaves in the trash, and does nothing when it is already
 * gone. It throws when the core refused the removal or the trashed copy could not be deleted.
 */
export const destroyNote = async (request: APIRequestContext, notePath: string): Promise<void> => {
  const answer = await callRPC(request, 'FileService/RemoveFile', { path: notePath })
  const error = answer['error']
  if (error && error !== 'ERROR_CODE_NOT_FOUND') {
    throw new Error(`RemoveFile ${notePath} refused: ${JSON.stringify(answer)}`)
  }
  const trashed = answer['trashed']
  if (typeof trashed !== 'string' || !trashed) return
  try {
    fs.rmSync(path.join(getVaultPath(), trashed), { recursive: true, force: true })
  } catch (failure) {
    throw new Error(`the trashed copy ${trashed} of ${notePath} stayed`, { cause: failure })
  }
}

const isIndexedNote = async (request: APIRequestContext, notePath: string): Promise<boolean> => {
  const answer = await callRPC(request, 'NoteService/ResolveAddresses', { written: [notePath] })
  const resolved = (answer['resolved'] as { path?: string }[] | undefined) ?? []
  return resolved.some((one) => one.path === notePath)
}

/**
 * Deletes a note file and returns once the index no longer holds its path, or at once when the
 * index did not hold it. The watcher reports the removal after RemoveFile returns, so a test that
 * counts the vectors next would count the vectors of the removed note too.
 */
export const destroyIndexedNote = async (
  request: APIRequestContext,
  notePath: string,
): Promise<void> => {
  const isIndexed = await isIndexedNote(request, notePath)
  await destroyNote(request, notePath)
  if (!isIndexed) return
  await expect.poll(() => isIndexedNote(request, notePath), { timeout: 45_000 }).toBe(false)
}

const findTrashedPaths = (name: string): string[] => {
  const vault = getVaultPath()
  const trash = path.join(vault, '.trash')
  if (!fs.existsSync(trash)) return []
  return fs
    .readdirSync(trash, { recursive: true, encoding: 'utf8' })
    .filter((one) => path.basename(one) === name)
    .map((one) => path.posix.join('.trash', one.split(path.sep).join('/')))
}

/** Takes the files of the vault's trash named `name` off the disk. */
export const destroyTrashed = (name: string): void => {
  const vault = getVaultPath()
  for (const one of findTrashedPaths(name)) fs.rmSync(path.join(vault, one))
}

/** One record a connection holds: the role it was written with, its type, label and marker. */
interface ConnectionRecord {
  readonly role?: string
  readonly type?: string
  readonly label?: string
}

/** The relationship between two notes, as the core reads it from both. */
interface Connection {
  readonly role?: string
  readonly firstSide?: ConnectionRecord
  readonly secondSide?: ConnectionRecord
}

/** The connection between two notes: what the second is to the first, and the records of each. */
export const getConnection = async (
  request: APIRequestContext,
  first: string,
  second: string,
): Promise<Connection> => {
  const answer = await callRPC(request, 'NoteService/GetConnection', { first, second })
  if (answer['error']) throw new Error(`GetConnection refused: ${JSON.stringify(answer)}`)
  return (answer['connection'] as Connection | undefined) ?? {}
}

/** The paths of the notes whose links point at the note. */
export const listBacklinks = async (
  request: APIRequestContext,
  notePath: string,
): Promise<string[]> => {
  const answer = await callRPC(request, 'NoteService/ListBacklinks', { path: notePath })
  const backlinks = (answer['backlinks'] as { from?: { path?: string } }[] | undefined) ?? []
  return backlinks.map((one) => one.from?.path ?? '')
}

/** One vault of the installation's list. */
interface Vault {
  readonly id: string
  readonly name: string
  readonly path: string
}

/** The vaults the installation lists. */
export const listVaults = async (request: APIRequestContext): Promise<Vault[]> => {
  const answer = await callRPC(request, 'VaultsService/ListVaults', {})
  return ((answer['vaults'] as Partial<Vault>[] | undefined) ?? []).map((one) => ({
    id: one.id ?? '',
    name: one.name ?? '',
    path: one.path ?? '',
  }))
}

/** The text of a file of the vault as it is on disk, frontmatter included. */
export const readVaultFile = (request: APIRequestContext, notePath: string): Promise<string> =>
  new Promise((resolve) => {
    resolve(fs.readFileSync(path.join(getVaultPath(), notePath), 'utf8'))
  })

/** Adds a folder of this machine to the installation's vaults, and gives back the vault. */
export const addVault = async (
  request: APIRequestContext,
  folder: string,
  name: string,
): Promise<Vault> => {
  const answer = await callRPC(request, 'VaultsService/AddVault', { path: folder, name })
  const vault = answer['vault'] as Partial<Vault> | undefined
  if (answer['error'] || !vault?.id) throw new Error(`AddVault refused: ${JSON.stringify(answer)}`)
  return { id: vault.id, name: vault.name ?? '', path: vault.path ?? '' }
}

/** Makes the vault the one the window shows. */
export const openVault = async (request: APIRequestContext, id: string): Promise<void> => {
  const answer = await callRPC(request, 'VaultsService/OpenVault', { id })
  if (answer['error']) throw new Error(`OpenVault refused: ${JSON.stringify(answer)}`)
}

/** Writes one setting of numen.json, the value given as JSON. */
export const writeSetting = async (
  request: APIRequestContext,
  keyPath: readonly string[],
  value: JsonValue | undefined,
): Promise<void> => {
  await callRPC(request, 'SettingsService/WriteSettings', {
    settings: [{ at: keyPath, value: JSON.stringify(value) }],
  })
}

const readSettings = async (request: APIRequestContext): Promise<JsonObject> =>
  JSON.parse(
    String((await callRPC(request, 'SettingsService/GetSettings', {}))['written'] ?? '{}'),
  ) as JsonObject

/** The values stored under an owner of numen.json: `app` or a plugin id. */
export const readOwner = async (request: APIRequestContext, owner: string): Promise<JsonObject> =>
  ((await readSettings(request))[owner] ?? {}) as JsonObject

/** A title no other test or earlier run uses, so a retry starts from a vault without it. */
export const createUniqueTitle = (words: string): string =>
  `${words} ${Math.random().toString(36).slice(2, 7)}`

/** The row of the files tab for the path, with the folder above it opened. */
export const revealTreeRow = async (page: Page, filePath: string): Promise<Locator> => {
  await page.getByRole('tab', { name: /^(Linked tab )?Files/ }).click()
  const folder = path.posix.dirname(filePath)
  if (folder !== '.') {
    const parent = page.locator(`[data-tree-row="${folder}"]`)
    await expect(parent).toBeVisible()
    if ((await parent.getAttribute('aria-expanded')) !== 'true') await parent.press('Enter')
  }
  const row = page.locator(`[data-tree-row="${filePath}"]`)
  await expect(row).toBeVisible()
  return row
}

/** The name a note tab is called by: the name of its file without the extension. */
export const getTabName = (notePath: string): string => path.posix.basename(notePath, '.md')

const getTabLabel = (notePath: string): string =>
  `${getTabName(notePath)} Close ${getTabName(notePath)}`

/** The tab of a note in the tab strip, of the note kind: a plex in focus on it is named alike. */
export const getNoteTab = (page: Page, notePath: string): Locator =>
  page
    .getByRole('tab', { name: getTabLabel(notePath), exact: true })
    .and(page.locator('[data-workspace-tab^="note:"]'))

/** The panel of the tab of a note, the one holding the note's text. */
export const getNotePanel = (page: Page, notePath: string): Locator =>
  page
    .getByRole('tabpanel', { name: getTabLabel(notePath), exact: true })
    .filter({ has: page.locator('.note__text') })

/** The editable text of the tab of a note. */
export const getNoteText = (page: Page, notePath: string): Locator =>
  getNotePanel(page, notePath).locator('.note__text .cm-content')

/** Opens a note of the vault from the files tab, and gives back the editable text of its tab. */
export const openNote = async (page: Page, notePath: string): Promise<Locator> => {
  await (await revealTreeRow(page, notePath)).press('Enter')
  const text = getNoteText(page, notePath)
  await expect(text).toBeVisible()
  await expect(text).toHaveAttribute('contenteditable', 'true')
  return text
}

/** Types at the end of the note in front. */
export const typeAtEnd = async (page: Page, text: Locator, typed: string): Promise<void> => {
  await text.click()
  await page.keyboard.press('ControlOrMeta+End')
  await page.keyboard.type(typed)
}
