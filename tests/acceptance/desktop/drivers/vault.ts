/** The vault of the scenario as a person finds it: what is in its folder before the application starts, and what is there after. */
import path from 'node:path'
import { expect } from 'playwright/test'
import {
  addVault,
  createNote,
  listFilePaths,
  listFiles,
  listVaults,
  openVault,
  waitForSavedBody,
  writeNote as writeBody,
} from '../../../e2e/desktop/tests/support/vault.js'
import {
  assertCardFileIsFree,
  assertCardFileKind,
  countCards,
  createCardId,
  createCardModelText,
  createCardText,
  createDeckText,
  createPresetText,
  createTemplatesText,
  getCardFile,
  parseCardModels,
  parseCards,
  parsePresets,
  readCardModelFields,
} from './cards.js'
import { addLinks, getLinkTitle, readLinks, type NoteLink } from './note-links.js'
import { assertEntry, DEFAULT_VAULT, getStem, parseColumn, readLinkAddress } from './rules.js'
import { assertPathsAreFree, parseVaultChecks, parseVaultFiles, type VaultWrite } from './vault-files.js'
import type { Scenario } from './scenario.js'

export class VaultDriver {
  constructor(private readonly scenario: Scenario) {}

  createEmptyVault(name: string): void {
    this.scenario.createVault(name)
  }

  async addFiles(vault: string, rows: string[][]): Promise<void> {
    const writes = parseVaultFiles(rows)
    if (!this.scenario.hasWorld) {
      this.scenario.checkSharedVault(vault)
      for (const write of writes) await this.writeSharedNote(write)
      return
    }
    const world = this.scenario.getWorld()
    const name = this.scenario.getVault(vault)
    assertPathsAreFree(writes, (entryPath) => world.getEntryKind(name, entryPath) !== undefined)
    for (const write of writes) await this.writeFile(name, write)
  }

  async expectContainsFiles(vault: string, rows: string[][]): Promise<void> {
    for (const check of parseVaultChecks(rows)) {
      await this.expectContains(vault, check.isFolder ? 'folder' : 'file', check.path)
      if (check.text !== '') await this.expectHasText(vault, check.path, check.text)
    }
  }

  addCardModels(vault: string, rows: string[][]): void {
    for (const model of parseCardModels(rows)) {
      this.writeNewCards(vault, model.title, createCardModelText(model.title, model.fields) + createTemplatesText(model.templates))
    }
  }

  addCards(vault: string, rows: string[][]): void {
    for (const card of parseCards(rows)) {
      this.writeDeck(vault, card.deck)
      if (card.model !== undefined) this.writeCard(vault, card.deck, card.model, card.values)
    }
  }

  addPresets(vault: string, rows: string[][]): void {
    for (const preset of parsePresets(rows)) {
      this.writeNewCards(vault, preset.title, createPresetText(preset.title, preset.settings))
      this.scenario.rememberNote(preset.title, getCardFile(preset.title))
    }
  }

  /** Makes the window show another vault of the world, over the calls of the vault list the palette command Open vault makes. */
  async switchTo(name: string): Promise<void> {
    const vault = this.scenario.getVault(name)
    const folder = this.scenario.getWorld().getFolder(vault)
    const { page } = this.scenario
    const vaults = await listVaults(page.request)
    const target = vaults.find((one) => one.path === folder) ?? (await addVault(page.request, folder, vault))
    await openVault(page.request, target.id)
    await expect(page.locator('[data-title]')).toContainText(vault)
  }

  async addToList(vault: string): Promise<void> {
    const name = this.scenario.getVault(vault)
    const folder = this.scenario.getWorld().getFolder(name)
    const { request } = this.scenario.page
    if (!(await listVaults(request)).some((one) => one.path === folder)) await addVault(request, folder, name)
  }

  /**
   * Declares the vaults of the list before the person has run a command, and checks the list after: the
   * list holds these vaults and no other.
   */
  async expectList(rows: readonly (readonly string[])[]): Promise<void> {
    const names = parseColumn(rows, 'vault')
    if (this.scenario.actions === 0) for (const name of names) await this.addToList(name)
    const { request } = this.scenario.page
    await expect
      .poll(async () => (await listVaults(request)).map((one) => one.name).sort())
      .toStrictEqual([...names].sort())
  }

  /** The links of the notes written before the application starts: each note of the column from writes its links. */
  addLinks(rows: readonly (readonly string[])[]): void {
    const world = this.scenario.getWorld()
    const vault = this.scenario.getVault(DEFAULT_VAULT)
    const links = parseLinkRows(rows, ['from', 'to', 'role'], ['description'])
    for (const from of new Set(links.map((link) => link.from))) {
      const notePath = this.scenario.getNotePath(from)
      const text = world.readText(vault, notePath)
      if (text === undefined) throw new Error(`no note is titled "${from}" in the vault: write it before its links`)
      const noteLinks: NoteLink[] = links
        .filter((link) => link.from === from)
        .map((link) => ({ to: `[[${this.scenario.getNotePath(link.to).replace(/\.md$/, '')}]]`, role: link.role, label: link.description }))
      world.writeText(vault, notePath, addLinks(text, noteLinks))
    }
  }

  /** The vault holds the links of the table and no other: a table of addresses lists link notes, a table of notes lists the links they write. */
  async expectLinks(vault: string, rows: readonly (readonly string[])[]): Promise<void> {
    if (rows[0]?.[0] === 'address') await this.expectLinkNotes(vault, rows)
    else await this.expectNoteLinks(vault, rows)
  }

  private async expectNoteLinks(vault: string, rows: readonly (readonly string[])[]): Promise<void> {
    const wanted = parseLinkRows(rows, ['from', 'to', 'role'], ['description'])
    const hasDescription = rows[0]?.includes('description') === true
    const world = this.scenario.getWorld()
    const name = this.scenario.getVault(vault)
    const format = (from: string, to: string, role: string, description: string): string =>
      [from, to, role, hasDescription ? description : ''].join(' | ')
    await expect
      .poll(() =>
        world
          .listNoteFiles(name)
          .flatMap((note) => readLinks(note.text).map((link) => format(getStem(note.path), getLinkTitle(link.to), link.role, link.label)))
          .sort(),
      )
      .toStrictEqual(wanted.map((link) => format(link.from, link.to, link.role, link.description)).sort())
  }

  private async expectLinkNotes(vault: string, rows: readonly (readonly string[])[]): Promise<void> {
    const addresses = parseColumn(rows, 'address')
    const world = this.scenario.getWorld()
    const name = this.scenario.getVault(vault)
    await expect
      .poll(() =>
        world
          .listNotes(name)
          .map((text) => readLinkAddress(text))
          .filter((one): one is string => one !== undefined)
          .sort(),
      )
      .toStrictEqual([...addresses].sort())
  }

  async expectWindowShows(name: string): Promise<void> {
    await expect(this.scenario.page.locator('[data-title]')).toContainText(name)
  }

  async expectContains(vault: string, entry: string, entryPath: string): Promise<void> {
    if (this.scenario.hasWorld) {
      const world = this.scenario.getWorld()
      const name = this.scenario.getVault(vault)
      await expect.poll(() => world.getEntryKind(name, entryPath)).toBeDefined()
      assertEntry(entry, entryPath, world.getEntryKind(name, entryPath) === 'folder')
      return
    }
    this.scenario.setVault(vault)
    await expect.poll(() => this.listEntries(entryPath)).toContain(entryPath)
    const parent = entryPath.split('/').slice(0, -1).join('/')
    const found = (await listFiles(this.scenario.page.request, parent)).find((one) => one.path === entryPath)
    assertEntry(entry, entryPath, found?.isFolder ?? false)
  }

  async expectDoesNotContain(vault: string, file: string): Promise<void> {
    if (this.scenario.hasWorld) {
      const world = this.scenario.getWorld()
      const name = this.scenario.getVault(vault)
      await expect.poll(() => world.getEntryKind(name, file)).toBeUndefined()
      return
    }
    this.scenario.setVault(vault)
    await expect.poll(() => this.listEntries(file)).not.toContain(file)
  }

  private async expectHasText(vault: string, notePath: string, text: string): Promise<void> {
    if (!this.scenario.hasWorld) {
      await waitForSavedBody(this.scenario.page.request, notePath, text)
      return
    }
    const world = this.scenario.getWorld()
    const name = this.scenario.getVault(vault)
    await expect.poll(() => world.readText(name, notePath), { message: `the note "${notePath}" does not have the text "${text}"` }).toContain(text)
  }

  private async writeFile(vault: string, write: VaultWrite): Promise<void> {
    const world = this.scenario.getWorld()
    if (write.kind === 'folder') world.makeFolder(vault, write.path)
    else if (write.kind === 'empty') world.writeText(vault, write.path, '')
    else if (write.kind === 'sample') world.copySample(vault, write.sampleKind, write.path, write.sample)
    else {
      world.writeNote(vault, write.path, write.text)
      this.scenario.rememberNote(getStem(write.path), write.path)
      if (this.scenario.hasServer) await this.expectListed(write.path)
    }
  }

  private async writeSharedNote(write: VaultWrite): Promise<void> {
    if (write.kind !== 'note') {
      throw new Error(`"${write.path}" is not a note: the vault without a name takes notes only, a world takes any file`)
    }
    const { request } = this.scenario.page
    const title = getStem(write.path)
    await this.scenario.claimTitle(title)
    const notePath = await createNote(request, title, path.posix.dirname(write.path).replace(/^\.$/, ''))
    if (notePath !== write.path) throw new Error(`the product made the note at "${notePath}", not at "${write.path}"`)
    this.scenario.rememberNote(title, notePath)
    await writeBody(request, notePath, `${write.text}\n`)
    await this.expectListed(notePath)
  }

  private writeDeck(vault: string, deck: string): void {
    const world = this.scenario.getWorld()
    const name = this.scenario.getVault(vault)
    const existing = world.readText(name, getCardFile(deck))
    if (existing === undefined) world.writeText(name, getCardFile(deck), createDeckText(deck))
    else assertCardFileKind(existing, deck, 'deck')
  }

  private writeCard(vault: string, deck: string, model: string, values: [string, string][]): void {
    const fields = readCardModelFields(this.readCards(vault, model, 'card model'))
    const text = this.readCards(vault, deck, 'deck')
    const card = createCardText(model, fields, values, createCardId(deck, countCards(text)))
    this.scenario.getWorld().writeText(this.scenario.getVault(vault), getCardFile(deck), text + card)
  }

  private readCards(vault: string, title: string, kind: 'card model' | 'deck'): string {
    return assertCardFileKind(this.scenario.getWorld().readText(this.scenario.getVault(vault), getCardFile(title)), title, kind)
  }

  private writeNewCards(vault: string, title: string, text: string): void {
    const world = this.scenario.getWorld()
    const name = this.scenario.getVault(vault)
    assertCardFileIsFree(name, title, world.readText(name, getCardFile(title)))
    world.writeText(name, getCardFile(title), text)
  }

  private async expectListed(filePath: string): Promise<void> {
    const parent = filePath.split('/').slice(0, -1).join('/')
    await expect.poll(() => listFilePaths(this.scenario.page.request, parent)).toContain(filePath)
  }

  private async listEntries(entryPath: string): Promise<string[]> {
    const parent = entryPath.split('/').slice(0, -1).join('/')
    return (await listFiles(this.scenario.page.request, parent)).map((one) => one.path)
  }
}

interface LinkRow {
  readonly from: string
  readonly to: string
  readonly role: string
  readonly description: string
}

const parseLinkRows = (rows: readonly (readonly string[])[], required: readonly string[], optional: readonly string[]): LinkRow[] => {
  const [header = [], ...body] = rows
  const unknown = header.filter((column) => ![...required, ...optional].includes(column))
  if (unknown.length > 0 || required.some((column) => !header.includes(column))) {
    throw new Error(`the table must have the header "| ${required.join(' | ')} |", and may add "${optional.join(', ')}"`)
  }
  const cell = (row: readonly string[], column: string): string => row[header.indexOf(column)] ?? ''
  return body.map((row) => ({ from: cell(row, 'from'), to: cell(row, 'to'), role: cell(row, 'role'), description: cell(row, 'description') }))
}
