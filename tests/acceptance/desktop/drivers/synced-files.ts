/** Files of any kind and size on the two computers, the breaks of the connection while they travel, and the answer to a paused removal. */
import path from 'node:path'
import { expect } from 'playwright/test'
import { callRPC } from '../../../e2e/desktop/tests/support/vault.js'
import type { ComputerDriver, ComputerName, Holder, Place } from './computer.js'
import { appendToFixture, buildFixture } from './file-fixtures.js'
import type { SyncDriver } from './sync.js'
import type { SyncVaultsDriver } from './sync-vaults.js'
import { getSampleKind } from './vault-files.js'

/** A file arrives within a few passes of the plugin on both computers; a big one takes longer. */
const ARRIVAL_LIMIT_MS = 90_000
/** The sample file a file is made from, by its extension: a person adds a real file, as the step "the vault has the following" does. */
const SAMPLE_OF_EXTENSION: Readonly<Record<string, string>> = { '.pdf': 'paper.pdf', '.png': 'llama.png' }
/** How many bytes are appended to a file a person changes. */
const CHANGE_BYTES = 100
/** The server breaks after a few parts of a big file went up, before the last: the parts are of 16 MiB. */
const BREAK_AFTER_BYTES = 20 * 1024 * 1024

export class SyncedFilesDriver {
  constructor(
    private readonly computers: ComputerDriver,
    private readonly sync: SyncDriver,
    private readonly vaults: SyncVaultsDriver,
  ) {}

  /** Puts a file in the vault: a sample when the file has an extension that has one and no size, else bytes of the size. */
  addFile(computer: ComputerName, filePath: string, megabytes: number | undefined, vault: string): void {
    const { world, vault: name } = this.computers.getPlace(computer, vault)
    const sample = SAMPLE_OF_EXTENSION[path.extname(filePath).toLowerCase()] as string | undefined
    if (megabytes === undefined && sample !== undefined) world.copySample(name, getSampleKind(sample), filePath, sample)
    else world.writeBytes(name, filePath, buildFixture(filePath, megabytes))
  }

  addFiles(computer: ComputerName, count: number, folder: string, vault: string): void {
    const { world, vault: name } = this.computers.getPlace(computer, vault)
    for (let number = 1; number <= count; number += 1) {
      const filePath = `${folder}/File ${number}.bin`
      world.writeBytes(name, filePath, buildFixture(filePath))
    }
  }

  async changeFile(computer: ComputerName, filePath: string, vault: string): Promise<void> {
    await this.sync.waitForInterruption()
    const { world, vault: name } = this.computers.getPlace(computer, vault)
    const content = world.readBytes(name, filePath)
    if (content === undefined) throw new Error(`the vault "${name}" has no file "${filePath}" to change`)
    world.writeBytes(name, filePath, appendToFixture(content, CHANGE_BYTES))
  }

  /** Moves a file or a folder as the person does in the application, so a note that moves has its links repaired. */
  async moveEntry(computer: ComputerName, kind: 'file' | 'folder', move: { from: string; to: string }, vault: string): Promise<void> {
    const { from, to } = move
    this.computers.getPlace(computer, vault)
    const answer = await callRPC(this.computers.getScenario(computer).page.request, 'FileService/MoveFile', { from, to })
    if (answer['error'] && answer['error'] !== 'ERROR_CODE_UNSPECIFIED') {
      throw new Error(`the ${kind} "${from}" was not moved to "${to}": ${JSON.stringify(answer)}`)
    }
  }

  /** Removes a file or a folder as the person does in the application: it goes to the trash of the vault. */
  async removeEntry(computer: ComputerName, kind: 'file' | 'folder', entryPath: string, vault: string): Promise<void> {
    this.computers.getPlace(computer, vault)
    const answer = await callRPC(this.computers.getScenario(computer).page.request, 'FileService/RemoveFile', { path: entryPath })
    if (answer['error'] && answer['error'] !== 'ERROR_CODE_UNSPECIFIED') {
      throw new Error(`the ${kind} "${entryPath}" was not removed: ${JSON.stringify(answer)}`)
    }
  }

  async expectSameFile(holder: Holder, filePath: string, vault: string | undefined): Promise<void> {
    const mine = this.computers.getPlace(holder.computer, holder.vault ?? vault)
    const other = this.computers.getPlace(holder.computer === 'first' ? 'second' : 'first', undefined)
    await this.expectSameBytes(mine, other, filePath)
  }

  async expectSameFileAsSecond(vault: string, filePath: string): Promise<void> {
    await this.expectSameBytes(this.computers.getPlace('first', vault), this.computers.getPlace('second', undefined), filePath)
  }

  async expectFileCount(holder: Holder, count: number, folder: string, vault: string | undefined): Promise<void> {
    const { world, vault: name } = this.computers.getPlace(holder.computer, holder.vault ?? vault)
    await expect.poll(() => world.countFiles(name, folder), { timeout: ARRIVAL_LIMIT_MS }).toBe(count)
  }

  /** Makes the sync server break once 20 MiB of the parts of the file went up; the step "the sync server runs again" ends the break. */
  async breakConnectionWhileSending(_filePath: string): Promise<void> {
    await this.sync.breakAfter('received', BREAK_AFTER_BYTES)
  }

  /** Makes the sync server break once 20 MiB of a blob went down to the second computer. */
  async breakConnectionWhileReceiving(_filePath: string): Promise<void> {
    await this.sync.breakAfter('sent', BREAK_AFTER_BYTES)
  }

  async choose(action: string): Promise<void> {
    await this.vaults.choose(action)
  }

  private async expectSameBytes(mine: Place, other: Place, filePath: string): Promise<void> {
    await expect
      .poll(
        () => {
          const have = mine.world.readBytes(mine.vault, filePath)
          const want = other.world.readBytes(other.vault, filePath)
          return have !== undefined && want !== undefined && have.equals(want)
        },
        { message: `the file "${filePath}" is the same on both computers`, timeout: ARRIVAL_LIMIT_MS },
      )
      .toBe(true)
  }
}
