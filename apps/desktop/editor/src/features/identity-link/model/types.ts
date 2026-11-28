/** What the devices of the identity need from the identity of this window. */
import type { Ref } from 'vue'
import type { Result } from '@numen/result'
import type { DeviceInfo } from '@/entities/identity'
import type { LinkCalls } from '../lib/types'

/** The devices of the identity, the revoke of one and the link of a new one. */
export interface DeviceCalls {
  readonly list: Readonly<Ref<readonly DeviceInfo[]>>
  load(): Promise<void>
  revoke(fingerprint: string): Promise<Result<undefined, string>>
  readonly link: LinkCalls
}
