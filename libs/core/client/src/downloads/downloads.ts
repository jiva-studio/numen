/**
 * How far the download of a file that is not on this device has come.
 */
import { ConnectError, type Client } from '@connectrpc/connect'
import type { FileService } from '@numen/protocol'
import { asFailure, asValue, type Result } from '@numen/result'

/** The call of FileService a window makes to ask how far a download has come. */
export type DownloadRpc = Pick<Client<typeof FileService>, 'getDownloadProgress'>

/** How far a download has come; `isKnown` is false when no download of the file runs. */
export interface DownloadProgress {
  readonly isKnown: boolean
  readonly bytesDone: number
  readonly bytesTotal: number
}

/** The downloads of the files that are not on this device, as a window asks about them. */
export interface DownloadClient {
  /** How many bytes of the file at `path` have come and how many the file has; the failure is the words the person reads. */
  getDownloadProgress(path: string): Promise<Result<DownloadProgress>>
}

/** The download calls of a window over `rpc`. */
export function createDownloadClient(rpc: DownloadRpc): DownloadClient {
  return {
    async getDownloadProgress(path) {
      try {
        const answer = await rpc.getDownloadProgress({ path })
        return asValue({
          isKnown: answer.isKnown,
          bytesDone: Number(answer.bytesDone),
          bytesTotal: Number(answer.bytesTotal),
        })
      } catch (failure) {
        return asFailure(ConnectError.from(failure).rawMessage)
      }
    },
  }
}
