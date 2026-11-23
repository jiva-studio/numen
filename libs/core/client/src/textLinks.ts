/**
 * The wikilinks of a text, resolved over the wire.
 *
 * Every window queries the core the same way, so the meaning of
 * `ResolveAddresses` in the port is written down once.
 */
import type { Client } from '@connectrpc/connect'
import type { NoteService } from '@numen/protocol'

/** What the core resolves for one text. Empty fields refer to the window's own. */
export interface TextLinkRequest {
  readonly text: string
  readonly from?: string | undefined
  readonly vault?: string | undefined
}

/** One wikilink of a text with the path it resolves to, empty where it resolves to none. */
export interface ResolvedTextLink {
  readonly raw: string
  readonly address: string
  readonly text: string
  readonly path: string
}

/** The call of NoteService the links are resolved through. */
export type TextLinkRpc = Pick<Client<typeof NoteService>, 'resolveAddresses'>

/**
 * The wikilinks of a text, each with the path it resolves to: empty where it
 * resolves to none, or to one in another vault.
 */
export const resolveTextLinks = async (
  client: TextLinkRpc,
  { text, from = '', vault = '' }: TextLinkRequest,
): Promise<readonly ResolvedTextLink[]> => {
  const answer = await client.resolveAddresses({ from, text, vaultId: vault })
  const paths = new Map(
    answer.resolvedAddresses.filter((one) => !one.isCrossed).map((one) => [one.address, one.path]),
  )
  return answer.links.map((one) => ({
    raw: one.raw,
    address: one.address,
    text: one.text,
    path: paths.get(one.address) ?? '',
  }))
}
