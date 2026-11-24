import type { Transport } from '@connectrpc/connect'
import { createFileRpc, createVaultRpc } from '@numen/core-client'
import type { Result } from '@numen/result'
import type { Core } from '../lib/core'

/** Where the sync plugin answers: the core's address and the token every request carries. */
export interface SyncEndpoint {
  readonly baseUrl: string
  readonly token: string
}

const SYNC_SERVICE = 'md.numen.tools.sync'

async function readFailure(answer: Response): Promise<string> {
  const failure: unknown = await answer.json().catch(() => null)
  const message =
    typeof failure === 'object' && failure !== null && 'message' in failure
      ? failure.message
      : undefined
  return typeof message === 'string' && message !== '' ? message : `The core answered ${answer.status}`
}

async function callSync(
  endpoint: SyncEndpoint,
  vaultId: string,
  method: 'FetchFile' | 'DropDownload',
  path: string,
): Promise<Result<null>> {
  const answer = await fetch(`${endpoint.baseUrl}/${SYNC_SERVICE}/${method}`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${endpoint.token}`,
      'Content-Type': 'application/json',
      'Numen-Vault': vaultId,
    },
    body: JSON.stringify({ vaultId, path }),
  })
  if (answer.ok) return { ok: true, value: null }
  return { ok: false, error: await readFailure(answer) }
}

/**
 * The calls of the file service over `transport`, and of the sync plugin at `sync`, in the window's own
 * types.
 */
export function createFileApi(
  transport: Transport,
  sync?: SyncEndpoint,
): Pick<Core, 'listFiles' | 'fetchFile' | 'dropDownload'> {
  const files = createFileRpc(transport)
  const vault = createVaultRpc(transport)
  const callPlugin = async (method: 'FetchFile' | 'DropDownload', path: string) => {
    if (!sync) return { ok: false as const, error: 'The sync is not on this phone' }
    return callSync(sync, (await vault.getVaultState({})).id, method, path)
  }
  return {
    async listFiles(folder) {
      const answer = await files.listFiles(folder === '' ? {} : { path: folder })
      return answer.entries.map((entry) => ({
        path: entry.path,
        name: entry.name,
        isFolder: entry.isFolder,
        isAbsent: entry.isAbsent,
      }))
    },
    fetchFile: (path) => callPlugin('FetchFile', path),
    dropDownload: (path) => callPlugin('DropDownload', path),
  }
}
