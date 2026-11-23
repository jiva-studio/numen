/** What the devices and the link of a device need from the identity of this window. */
import type { Result } from '@numen/result'
import type { LinkInfo, LinkMode, LinkRole } from '@/entities/identity'

/** The link of a device to the account by a QR code, as the dialog drives it. */
export interface LinkCalls {
  start(role: LinkRole, mode: LinkMode, scannedText: string): Promise<Result<LinkInfo, string>>
  get(): Promise<Result<LinkInfo, string>>
  decide(approve: boolean): Promise<Result<undefined, string>>
  enterCode(code: string): Promise<Result<undefined, string>>
  confirmJoin(approve: boolean): Promise<Result<undefined, string>>
  cancel(): Promise<Result<undefined, string>>
}

/** A labelled value of a link: a name, a fingerprint, a code. */
export interface Fact {
  label: string
  value: string
  isMono: boolean
}
