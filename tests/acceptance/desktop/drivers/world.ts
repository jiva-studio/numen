/** A world: a folder of one scenario that holds one folder for each vault, named by the vault. */
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { getPdfSample } from '../support/samples.js'
import { assertVaultName, assertVaultPath, getNoteFile, pickVault } from './rules.js'

export type EntryKind = 'file' | 'folder'

export class World {
  private readonly vaults: string[] = []

  private constructor(readonly root: string) {}

  static create(): World {
    return new World(fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), 'numen-world-'))))
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
    this.writeFile(vault, getNoteFile(title), `${text}\n`)
  }

  makeFolder(vault: string, folder: string): void {
    assertVaultPath(folder)
    fs.mkdirSync(path.join(this.getFolder(vault), folder), { recursive: true })
  }

  copyPdf(vault: string, filePath: string): void {
    assertVaultPath(filePath)
    const target = path.join(this.getFolder(vault), filePath)
    fs.mkdirSync(path.dirname(target), { recursive: true })
    fs.copyFileSync(getPdfSample(path.basename(filePath)), target)
  }

  getEntryKind(vault: string, entryPath: string): EntryKind | undefined {
    assertVaultPath(entryPath)
    const stat = fs.statSync(path.join(this.getFolder(vault), entryPath), { throwIfNoEntry: false })
    if (stat === undefined) return undefined
    return stat.isDirectory() ? 'folder' : 'file'
  }

  remove(): void {
    fs.rmSync(this.root, { recursive: true, force: true })
  }

  private writeFile(vault: string, filePath: string, text: string): void {
    const target = path.join(this.getFolder(vault), filePath)
    fs.mkdirSync(path.dirname(target), { recursive: true })
    fs.writeFileSync(target, text)
  }
}
