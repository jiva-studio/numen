/** The person who shares folders with readers: the dialog of the owner, the dialogs of the readers and the messages the owner receives. */
import { expect } from 'playwright/test'
import type { IdentityDriver } from './identity.js'
import type { ShareDriver } from './share.js'
import type { ReaderName, SharedReaderDriver } from './shared-reader.js'

export class SharingDriver {
  private link: string | undefined
  private refusal: string | undefined
  private fingerprint: string | undefined

  constructor(
    private readonly share: ShareDriver,
    private readonly readers: SharedReaderDriver,
    private readonly identity: IdentityDriver,
  ) {}

  /**
   * The person opens the dialog of the folder and copies the link, the reader pastes it in its dialog and checks the
   * owner, the reader tells the code it sees, and the person allows the device that shows the same code. A dialog that
   * refuses stops there and shows the reason.
   */
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

  async shareFolder(folder: string, reader: ReaderName): Promise<void> {
    const code = await this.ask(folder, reader)
    if (code === undefined) return
    await this.share.allow(this.getReaderFingerprint(), code)
    await this.readers.expectReceived(reader)
  }

  async declineFolder(folder: string, reader: ReaderName): Promise<void> {
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

  async expectTold(text: string): Promise<void> {
    if (this.refusal !== undefined) expect(this.refusal).toContain(text)
    else await this.share.expectTold(text)
  }
}
