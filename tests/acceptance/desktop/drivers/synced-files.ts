/** Files of any kind and size on the two computers, the breaks of the sync server while they travel, and the answer to a paused removal. */
import type { ComputerName, Holder } from './computer.js'

const notImplemented = (what: string): never => {
  throw new Error(`not implemented: ${what}`)
}

export class SyncedFilesDriver {
  writeFile(computer: ComputerName, filePath: string, megabytes: number | undefined, vault: string): void {
    notImplemented(`write ${filePath} of ${megabytes ?? 'default'} MB in ${vault} of ${computer}`)
  }

  writeFiles(computer: ComputerName, count: number, folder: string, vault: string): void {
    notImplemented(`write ${count} files in ${folder} of ${vault} of ${computer}`)
  }

  changeFile(computer: ComputerName, filePath: string, vault: string): void {
    notImplemented(`change ${filePath} in ${vault} of ${computer}`)
  }

  moveEntry(computer: ComputerName, kind: 'file' | 'folder', from: string, to: string, vault: string): void {
    notImplemented(`move the ${kind} ${from} to ${to} in ${vault} of ${computer}`)
  }

  removeEntry(computer: ComputerName, kind: 'file' | 'folder', entryPath: string, vault: string): void {
    notImplemented(`remove the ${kind} ${entryPath} in ${vault} of ${computer}`)
  }

  async expectSameFile(holder: Holder, filePath: string, vault: string | undefined): Promise<void> {
    await Promise.reject(new Error(`not implemented: ${holder.computer} has the same file ${filePath} in ${vault ?? 'its vault'}`))
  }

  async expectSameFileAsSecond(vault: string, filePath: string): Promise<void> {
    await Promise.reject(new Error(`not implemented: ${vault} has the same file ${filePath} as the second computer`))
  }

  async expectFileCount(holder: Holder, count: number, folder: string, vault: string | undefined): Promise<void> {
    await Promise.reject(new Error(`not implemented: ${holder.computer} has ${count} files in ${folder} of ${vault ?? 'its vault'}`))
  }

  async expectNeverBothFolders(first: string, second: string): Promise<void> {
    await Promise.reject(new Error(`not implemented: the second computer never showed files in both ${first} and ${second}`))
  }

  async stopServerAfterBytes(megabytes: number, filePath: string): Promise<void> {
    await Promise.reject(new Error(`not implemented: the server stops after ${megabytes} MB of ${filePath}`))
  }

  async stopServerWhileLoading(filePath: string): Promise<void> {
    await Promise.reject(new Error(`not implemented: the server stops while the second computer loads ${filePath}`))
  }

  async expectNoRepeatedPart(filePath: string, direction: 'received' | 'sent'): Promise<void> {
    await Promise.reject(new Error(`not implemented: the server ${direction} no part of ${filePath} twice`))
  }

  async choose(action: string): Promise<void> {
    await Promise.reject(new Error(`not implemented: the person chooses ${action}`))
  }
}
