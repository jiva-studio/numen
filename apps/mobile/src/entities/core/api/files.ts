import { ConnectError, type Transport } from '@connectrpc/connect'
import { createFileRpc } from '@numen/core-client'
import type { Result } from '@numen/result'
import type { Core } from '../lib/core'

async function run(call: () => Promise<unknown>): Promise<Result<null>> {
  try {
    await call()
    return { ok: true, value: null }
  } catch (failure) {
    return { ok: false, error: ConnectError.from(failure).rawMessage }
  }
}

/** The calls of the file service over `transport`, in the window's own types. */
export function createFileApi(
  transport: Transport,
): Pick<Core, 'listFiles' | 'fetchFile' | 'dropDownload'> {
  const files = createFileRpc(transport)
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
    fetchFile: (path) => run(() => files.fetchFile({ path })),
    dropDownload: (path) => run(() => files.dropDownload({ path })),
  }
}
