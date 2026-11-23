/** What the devices and the link of a device need from the identity of this window. */
import type { Ref } from 'vue'
import type { Result } from '@numen/result'
import type { DeviceInfo, LinkInfo, LinkMode, LinkRole } from '@/entities/identity'

/** The link of a device to the account by a QR code, as the dialog drives it. */
export interface LinkCalls {
  start(role: LinkRole, mode: LinkMode, scannedText: string): Promise<Result<LinkInfo, string>>
  get(): Promise<Result<LinkInfo, string>>
  decide(approve: boolean): Promise<Result<undefined, string>>
  enterCode(code: string): Promise<Result<undefined, string>>
  confirmJoin(approve: boolean): Promise<Result<undefined, string>>
  cancel(): Promise<Result<undefined, string>>
}

/** The devices of the account, the revoke of one and the link of a new one. */
export interface DeviceCalls {
  readonly devices: Readonly<Ref<readonly DeviceInfo[]>>
  loadDevices(): Promise<void>
  revoke(fingerprint: string): Promise<Result<undefined, string>>
  readonly link: LinkCalls
}

/** A labelled value of a link: a name, a fingerprint, a code. */
export interface Fact {
  label: string
  value: string
  isMono: boolean
}
