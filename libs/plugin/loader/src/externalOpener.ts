/**
 * Opening an address from a plugin.
 *
 * A plugin's `openExternal` passes an http or https address to this and fails any other address
 * before the window sees it. The window opens the address in the system browser.
 */
import { asFailure, asValue, formatErrorMessage, type Result } from '@numen/plugin-sdk'

/** Opens an address in the system browser. */
export type ExternalOpener = (url: string) => Promise<void>

/** Whether an address is an absolute http or https address. */
export function isWebAddress(url: string): boolean {
  try {
    const { protocol } = new URL(url)
    return protocol === 'http:' || protocol === 'https:'
  } catch {
    return false
  }
}

export function createExternalOpener(open: ExternalOpener): (url: string) => Promise<Result<null>> {
  return async (url) => {
    if (!isWebAddress(url)) return asFailure(`not a web address: ${url}`)
    try {
      await open(url)
      return asValue(null)
    } catch (error) {
      return asFailure(formatErrorMessage(error))
    }
  }
}
