/** The devices of the identity and the link of a new one, as the core holds them. */
import { shallowRef } from 'vue'
import type { Result } from '@numen/result'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { DevicesApi, LinkApi } from '../api/identity'
import type { DeviceInfo, IdentityFailure, LinkInfo, LinkMode, LinkRole } from '../lib/types'
import { IDENTITY_WORDS } from '../words'

interface DevicesCalls {
  readonly devices: DevicesApi
  readonly link: LinkApi
}

export function createDeviceStore(api: DevicesCalls, notify: MessageWriter) {
  const list = shallowRef<readonly DeviceInfo[]>([])

  const load = async (): Promise<void> => {
    try {
      const answer = await api.devices.listDevices()
      if (answer.ok) list.value = answer.value
    } catch (caughtError) {
      notify(`${IDENTITY_WORDS.loadFailed} ${formatError(caughtError)}`, 'error')
    }
  }

  /** Runs a call that answers with a refusal or a value and returns the refusal as text. */
  const ask = async <T>(
    call: () => Promise<Result<T, IdentityFailure>>,
  ): Promise<Result<T, string>> => {
    try {
      const answer = await call()
      return answer.ok ? answer : { ok: false, error: IDENTITY_WORDS.describeFailure(answer.error) }
    } catch (caughtError) {
      return { ok: false, error: formatError(caughtError) }
    }
  }

  const link = {
    start: (role: LinkRole, mode: LinkMode, text: string): Promise<Result<LinkInfo, string>> =>
      ask(() => api.link.startLink(role, mode, text)),
    get: (): Promise<Result<LinkInfo, string>> => ask(() => api.link.getLink()),
    decide: (approve: boolean) => ask(() => api.link.decideLink(approve)),
    confirmJoin: (approve: boolean) => ask(() => api.link.confirmJoin(approve)),
    cancel: () => ask(() => api.link.cancelLink()),
  }

  const revoke = async (fingerprint: string): Promise<Result<undefined, string>> => {
    const answer = await ask(() => api.devices.revokeDevice(fingerprint))
    await load()
    return answer
  }

  return { list, load, revoke, link }
}
