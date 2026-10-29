/**
 * The wikilinks of a text, resolved over the wire.
 *
 * Every window queries the core the same way, so the meaning of
 * `ResolveAddresses` in the port is written down once.
 */
import type { TextLink } from '../agent'

/** What the core is asked about one text. Empty fields refer to the window's own. */
export interface LinkRequest {
  readonly text: string
  /** The note the text is written in; a name resolves by a path relative to it. */
  readonly from?: string | undefined
  /** The vault the text is written in. */
  readonly vault?: string | undefined
}

/** As much of the note service as link resolution requires. */
export interface LinkClient {
  resolveAddresses(request: { from: string; text: string; vault: string }): Promise<{
    readonly found: readonly {
      readonly raw: string
      readonly address: string
      readonly text: string
    }[]
    readonly resolved: readonly {
      readonly written: string
      readonly path: string
      readonly isCrossed: boolean
    }[]
  }>
}

/**
 * The wikilinks of a text, each with the path it resolves to: empty where it
 * resolves to none, or to one in another vault.
 */
export const resolveTextLinks = async (
  client: LinkClient,
  { text, from = '', vault = '' }: LinkRequest,
): Promise<readonly TextLink[]> => {
  const answer = await client.resolveAddresses({ from, text, vault })
  const paths = new Map(
    answer.resolved.filter((one) => !one.isCrossed).map((one) => [one.written, one.path]),
  )
  return answer.found.map((one) => ({
    raw: one.raw,
    address: one.address,
    text: one.text,
    path: paths.get(one.address) ?? '',
  }))
}
