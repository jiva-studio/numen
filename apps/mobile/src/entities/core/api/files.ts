import type { Transport } from '@connectrpc/connect'
import type { Core } from '../lib/core'

/** The calls of the file service and of the sync plugin over `transport`, in the window's own types. */
export function createFileApi(
  transport: Transport,
): Pick<Core, 'listFiles' | 'fetchFile' | 'dropDownload'> {
  void transport
  const notImplemented = () => Promise.reject(new Error('not implemented'))
  return {
    listFiles: notImplemented,
    fetchFile: notImplemented,
    dropDownload: notImplemented,
  }
}
