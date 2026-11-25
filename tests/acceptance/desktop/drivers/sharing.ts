/** The person who shares folders with readers: the dialog of the owner, the dialogs of the readers and the messages the owner receives. */
import { expect } from 'playwright/test'
import type { IdentityDriver } from './identity.js'
import type { Scenario } from './scenario.js'
import type { ShareDriver } from './share.js'
import type { ReaderName, SharedReaderDriver } from './shared-reader.js'

export class SharingDriver {
  private link: string | undefined
  private refusal: string | undefined
  private fingerprint: string | undefined

  constructor(
    private readonly scenario: Scenario,
    private readonly share: ShareDriver,
    private readonly readers: SharedReaderDriver,
    private readonly identity: IdentityDriver,
  ) {}

  private async ask(folder: string, reader: ReaderName): Promise<string | undefined> {
    this.refusal = undefined
    const started = await this.share.start(folder)
    if ('refusal' in started) {
      this.refusal = started.refusal
      await this.share.close()
      return undefined
    }
    this.link = started.link
    this.fingerprint = this.readers.getFingerprint(reader)
    return this.readers.ask(reader, started.link, this.identity.getAccountFingerprint())
  }

  private requireFolder(vault: string, folder: string): void {
    const name = this.scenario.getVault(vault)
    if (folder !== '') expect(this.scenario.getWorld().getEntryKind(name, folder), `the vault "${name}" has the folder "${folder}"`).toBe('folder')
  }

  async shareFolder(vault: string, folder: string, reader: ReaderName): Promise<void> {
    this.requireFolder(vault, folder)
    const code = await this.ask(folder, reader)
    if (code === undefined) return
    await this.share.allow(this.getReaderFingerprint(), code)
    await this.readers.expectReceived(reader)
  }

  async declineFolder(vault: string, folder: string, reader: ReaderName): Promise<void> {
    this.requireFolder(vault, folder)
    const code = await this.ask(folder, reader)
    if (code === undefined) return
    await this.share.decline(this.getReaderFingerprint(), code)
  }

  async giveLinkTo(reader: ReaderName): Promise<void> {
    if (this.link === undefined) throw new Error('no link was given: begin with the person shares the folder')
    await this.readers.tryAccept(reader, this.link)
  }

  private getReaderFingerprint(): string {
    if (this.fingerprint === undefined) throw new Error('no reader was asked: begin with the person shares the folder')
    return this.fingerprint
  }

  async expectReaderListed(folder: string): Promise<void> {
    await this.share.expectReaderListed(folder, this.getReaderFingerprint())
  }

  async removeReader(folder: string): Promise<void> {
    await this.share.removeReader(folder, this.getReaderFingerprint())
  }

  /** Whether the share was refused with words that have the text. */
  isRefusedWith(text: string): boolean {
    return this.refusal?.includes(text) === true
  }
}
