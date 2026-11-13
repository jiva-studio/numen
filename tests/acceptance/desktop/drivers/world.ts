/** A world: a folder of one scenario that holds one folder for each vault, named by the vault. */
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { readSample, type SampleKind } from '../support/samples.js'
import { freeFromRemoval, holdForRemoval } from './reaper.js'
import { assertVaultName, assertVaultPath, pickVault } from './rules.js'

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

  writeNote(vault: string, filePath: string, text: string): void {
    this.writeText(vault, filePath, `${text}\n`)
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

  /** Every note of the vault, found below its folders, by its path and its text; the folders a program keeps (their names begin with a dot) are left out. */
  listNoteFiles(vault: string): { path: string; text: string }[] {
    const root = this.getFolder(vault)
    const walk = (folder: string): { path: string; text: string }[] =>
      fs.readdirSync(folder, { withFileTypes: true }).flatMap((entry) => {
        const full = path.join(folder, entry.name)
        if (entry.isDirectory()) return entry.name.startsWith('.') ? [] : walk(full)
        if (!entry.name.endsWith('.md')) return []
        return [{ path: path.relative(root, full).split(path.sep).join('/'), text: fs.readFileSync(full, 'utf8') }]
      })
    return walk(root)
  }

  /** The text of every note of the vault. */
  listNotes(vault: string): string[] {
    return this.listNoteFiles(vault).map((note) => note.text)
  }

  makeFolder(vault: string, folder: string): void {
    assertVaultPath(folder)
    fs.mkdirSync(path.join(this.getFolder(vault), folder), { recursive: true })
  }

  /** Puts the sample file of a kind, found by its file name, in the vault at the path. */
  copySample(vault: string, kind: SampleKind, filePath: string, fileName: string): void {
    assertVaultPath(filePath)
    const target = path.join(this.getFolder(vault), filePath)
    fs.mkdirSync(path.dirname(target), { recursive: true })
    fs.writeFileSync(target, readSample(kind, fileName))
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
