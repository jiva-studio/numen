/** The vault of the scenario as a person finds it: what is in its folder before the application starts, and what is there after. */
import { expect } from 'playwright/test'
import {
  addVault,
  createNote,
  listFilePaths,
  listFiles,
  listVaults,
  openVault,
  writeNote as writeBody,
} from '../../../e2e/desktop/tests/support/vault.js'
import {
  assertCardFileIsFree,
  assertCardFileKind,
  assertTemplatesAreNew,
  countCards,
  createCardId,
  createCardModelText,
  createCardText,
  createDeckText,
  createPresetText,
  createTemplatesText,
  getCardFile,
  parseFieldNames,
  parsePresetSettings,
  parseTemplates,
  parseValues,
  readCardModelFields,
} from './cards.js'
import { assertEntry, getNoteFile } from './rules.js'
import type { SampleKind } from '../support/samples.js'
import type { Scenario } from './scenario.js'

export class VaultDriver {
  constructor(private readonly scenario: Scenario) {}

  createEmpty(name: string): void {
    this.scenario.createVault(name)
  }

  async writeNote(vault: string, title: string, text: string): Promise<void> {
    if (this.scenario.hasWorld) {
      this.scenario.getWorld().writeNote(this.scenario.getVault(vault), title, text)
      this.scenario.rememberNote(title, getNoteFile(title))
      if (this.scenario.hasServer) await this.expectListed(getNoteFile(title))
      return
    }
    this.scenario.checkSharedVault(vault)
    const { request } = this.scenario.page
    await this.scenario.claimTitle(title)
    const notePath = await createNote(request, title)
    this.scenario.rememberNote(title, notePath)
    await writeBody(request, notePath, `${text}\n`)
    await this.expectListed(notePath)
  }

  makeFolder(vault: string, folder: string): void {
    this.scenario.getWorld().makeFolder(this.scenario.getVault(vault), folder)
  }

  copySample(vault: string, kind: SampleKind, filePath: string): void {
    this.scenario.getWorld().copySample(this.scenario.getVault(vault), kind, filePath)
  }

  addCardModel(vault: string, title: string, fields: string[][]): void {
    this.writeNewCards(vault, title, createCardModelText(title, parseFieldNames(fields)))
  }

  addTemplates(vault: string, title: string, templates: string[][]): void {
    const text = this.readCards(vault, title, 'card model')
    assertTemplatesAreNew(text, title)
    this.scenario.getWorld().writeText(this.scenario.getVault(vault), getCardFile(title), text + createTemplatesText(parseTemplates(templates)))
  }

  addDeck(vault: string, title: string): void {
    this.writeNewCards(vault, title, createDeckText(title))
  }

  addCard(vault: string, model: string, deck: string, values: string[][]): void {
    const fields = readCardModelFields(this.readCards(vault, model, 'card model'))
    const text = this.readCards(vault, deck, 'deck')
    const card = createCardText(model, fields, parseValues(values), createCardId(deck, countCards(text)))
    this.scenario.getWorld().writeText(this.scenario.getVault(vault), getCardFile(deck), text + card)
  }

  addPreset(vault: string, title: string, settings: string[][]): void {
    this.writeNewCards(vault, title, createPresetText(title, parsePresetSettings(settings)))
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
