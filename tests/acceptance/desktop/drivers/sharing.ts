/** The person who shares folders with readers: the dialog of the owner, the pages of the readers and the messages the owner receives. */
import { expect } from 'playwright/test'
import type { IdentityDriver } from './identity.js'
import type { ShareDriver, GivenInvitation } from './share.js'
import type { ReaderName, SharedReaderDriver } from './shared-reader.js'

export class SharingDriver {
  private invitation: GivenInvitation | undefined
  private refusal: string | undefined
  private fingerprint: string | undefined

  constructor(
    private readonly share: ShareDriver,
    private readonly readers: SharedReaderDriver,
    private readonly identity: IdentityDriver,
  ) {}

  /**
   * The person copies the code of the reader, adds the reader in the dialog, compares the fingerprint of the reader on both
   * sides, copies the invitation and the reader receives it. A dialog that refuses stops there and shows the reason.
   */
  async shareFolder(folder: string, reader: ReaderName): Promise<void> {
    this.refusal = undefined
    const outcome = await this.share.addReader(folder, await this.readers.getCode(reader), this.readers.getFingerprint(reader), reader)
    if ('refusal' in outcome) {
      this.refusal = outcome.refusal
      return
    }
    this.invitation = outcome.invitation
    this.fingerprint = outcome.invitation.readerFingerprint
    await this.readers.receive(reader, outcome.invitation, this.identity.getAccountFingerprint())
  }

  async giveInvitationTo(reader: ReaderName): Promise<void> {
    if (this.invitation === undefined) throw new Error('no invitation was given: begin with the person shares the folder')
    await this.readers.tryReceive(reader, this.invitation)
  }

  async expectReaderListed(folder: string): Promise<void> {
    if (this.fingerprint === undefined) throw new Error('no reader was added: begin with the person shares the folder')
    await this.share.expectReaderListed(folder, this.fingerprint)
  }

  async removeReader(folder: string): Promise<void> {
    if (this.fingerprint === undefined) throw new Error('no reader was added: begin with the person shares the folder')
    await this.share.removeReader(folder, this.fingerprint)
  }

  async expectTold(text: string): Promise<void> {
    if (this.refusal !== undefined) expect(this.refusal).toContain(text)
    else await this.share.expectTold(text)
  }
}
