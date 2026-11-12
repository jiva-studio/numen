/**
 * Address calls over the window's transport.
 */
import { createClient } from '@connectrpc/connect'
import { MediaService } from '@numen/protocol'
import { retryWhileBusy } from './errors/answers'
import { transport } from './transport'

const mediaClient = createClient(MediaService, transport)

/** Where the bytes at an address are loaded from, and their content type; both empty where unknown. */
export interface AddressSource {
  readonly url: string
  readonly mediaType: string
}

/**
 * Requests from the host where the file at an address is loaded from: a path of the vault, or
 * `plugin://<plugin id>/<path>` for a file in that plugin's store of the vault.
 */
export async function resolveAddress(address: string): Promise<AddressSource> {
  const answer = await retryWhileBusy(() => mediaClient.resolveAddress({ address }))
  return { url: answer.src, mediaType: answer.mediaType }
}
