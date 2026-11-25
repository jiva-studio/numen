/**
 * The JSON messages of the identity methods the plugin's backend serves, and the client that calls them. Keys and
 * meaning match the Go messages in backend/api; testdata/api holds the fixtures both sides are tested against.
 */
import { ReportedError, type Api, type ApiError } from '@numen/plugin-sdk'

import { REQUIREMENT_UNMET } from './call'

/**
 * The identity fingerprint is the name of the identity, which the person types to confirm a deletion. The time is
 * RFC 3339 and empty unless the account is being deleted. A device with no identity has neither.
 */
interface AccountStateResponse {
  readonly hasIdentity: boolean
  readonly account: string
  readonly isDeleting: boolean
  readonly deleteAt: string
}

interface DeleteCloudDataResponse {
  readonly deleteAt: string
}

interface ExportAccountDataResponse {
  readonly document: string
}

export interface AccountClient {
  getAccountState(): Promise<AccountStateResponse>
  /** When the server deletes the account for good, an RFC 3339 time. */
  deleteCloudData(): Promise<string>
  restoreAccount(): Promise<void>
  /** The JSON document of what the server holds of the identity. */
  exportAccountData(): Promise<string>
}

const NO_IDENTITY: AccountStateResponse = {
  hasIdentity: false,
  account: '',
  isDeleting: false,
  deleteAt: '',
}

const createFailure = (error: ApiError): Error =>
  Object.assign(new ReportedError(error.message), { code: error.code })

export function createAccountClient(api: Api): AccountClient {
  async function call<Req extends object, Res>(
    method: string,
    request: Req,
    ifNoIdentity?: Res,
  ): Promise<Res> {
    const result = await api.call<Req, Res>(method, request)
    if (!result.ok && ifNoIdentity !== undefined && result.error.code === REQUIREMENT_UNMET) {
      return ifNoIdentity
    }
    if (!result.ok) throw createFailure(result.error)
    return result.value
  }

  return {
    getAccountState: () =>
      call<Record<string, never>, AccountStateResponse>('GetAccountState', {}, NO_IDENTITY),
    deleteCloudData: async () =>
      (await call<Record<string, never>, DeleteCloudDataResponse>('DeleteCloudData', {})).deleteAt,
    restoreAccount: async () => {
      await call<Record<string, never>, Record<string, never>>('RestoreAccount', {})
    },
    exportAccountData: async () =>
      (await call<Record<string, never>, ExportAccountDataResponse>('ExportAccountData', {}))
        .document,
  }
}
