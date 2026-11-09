/** A world: a folder of one scenario that holds one folder for each vault, named by the vault. */
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { readSample, type SampleKind } from '../support/samples.js'
import { freeFromRemoval, holdForRemoval } from './reaper.js'
import { assertVaultName, assertVaultPath, getNoteFile, pickVault } from './rules.js'

export type EntryKind = 'file' | 'folder'

export class World {
  private readonly vaults: string[] = []
  private readonly pageTexts = new Map<string, string>()
  private hasNote = false

  private constructor(readonly root: string) {}

  static create(): World {
    const root = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), 'numen-world-')))
    holdForRemoval('dir', root)
    return new World(root)
  }

  /** Whether a step wrote a note: the window opens on the plex of the note the vault changed last. */
  hasNotes(): boolean {
    return this.hasNote
  }

  /** Makes the page of the address answer with the text in its first paragraph, for the server of this world. */
  setPageText(address: string, text: string): void {
    this.pageTexts.set(address, text)
  }

  readPageText(address: string): string | undefined {
    return this.pageTexts.get(address)
  }

  getVaults(): readonly string[] {
    return this.vaults
  }

  addVault(name: string): void {
    assertVaultName(name)
    if (this.vaults.includes(name)) throw new Error(`the world has the vault "${name}" already`)
    fs.mkdirSync(path.join(this.root, name))
    this.vaults.push(name)
  }

  getFolder(name: string): string {
    return path.join(this.root, pickVault(this.vaults, name))
  }

  writeNote(vault: string, title: string, text: string): void {
    this.writeText(vault, getNoteFile(title), `${text}\n`)
    this.hasNote = true
  }

  /** Writes a file of the vault at the path, with the folders above it. */
  writeText(vault: string, filePath: string, text: string): void {
    assertVaultPath(filePath)
    const target = path.join(this.getFolder(vault), filePath)
    fs.mkdirSync(path.dirname(target), { recursive: true })
    fs.writeFileSync(target, text)
  }

  readText(vault: string, filePath: string): string | undefined {
    assertVaultPath(filePath)
    const target = path.join(this.getFolder(vault), filePath)
    return fs.existsSync(target) ? fs.readFileSync(target, 'utf8') : undefined
  }

  makeFolder(vault: string, folder: string): void {
    assertVaultPath(folder)
    fs.mkdirSync(path.join(this.getFolder(vault), folder), { recursive: true })
  }

  /** Puts the sample file of a kind in the vault at the path, which names the sample by its file name. */
  copySample(vault: string, kind: SampleKind, filePath: string): void {
    assertVaultPath(filePath)
    const target = path.join(this.getFolder(vault), filePath)
    fs.mkdirSync(path.dirname(target), { recursive: true })
    fs.writeFileSync(target, readSample(kind, path.basename(filePath)))
  }

  getEntryKind(vault: string, entryPath: string): EntryKind | undefined {
    assertVaultPath(entryPath)
    const stat = fs.statSync(path.join(this.getFolder(vault), entryPath), { throwIfNoEntry: false })
    if (stat === undefined) return undefined
    return stat.isDirectory() ? 'folder' : 'file'
  }

  remove(): void {
    fs.rmSync(this.root, { recursive: true, force: true })
    freeFromRemoval('dir', this.root)
  }
}
