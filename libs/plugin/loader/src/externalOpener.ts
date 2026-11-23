/**
 * Opening an address from a plugin.
 *
 * A plugin's `openExternal` passes an http or https address to this and fails any other address
 * before the window sees it. The window opens the address in the system browser.
 */
import { isWebAddress, type OpenExternalErrorCode } from '@numen/plugin-sdk'
import { asFailure, asValue, formatErrorMessage, type Failure, type Result } from '@numen/result'

/** Opens an address in the system browser. */
export type ExternalOpener = (url: string) => Promise<void>

export function createExternalOpener(
  open: ExternalOpener,
): (url: string) => Promise<Result<null, Failure<OpenExternalErrorCode>>> {
  return async (url) => {
    if (!isWebAddress(url)) {
      return asFailure({ error: 'notWebAddress', message: `not a web address: ${url}` })
    }
    try {
      await open(url)
      return asValue(null)
    } catch (error) {
      return asFailure({ error: 'openFailed', message: formatErrorMessage(error) })
    }
  }
}
