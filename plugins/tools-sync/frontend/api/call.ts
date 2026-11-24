import type { Api } from '@numen/plugin-sdk'

export type Empty = Record<string, never>

/** The code of a call the host refused: the device has no identity, so the backend does not run. */
export const REQUIREMENT_UNMET = 'requirement_unmet'

/** Calls a method of the plugin's backend and throws the message of a failure. */
export async function call<Req, Res>(api: Api, method: string, request: Req): Promise<Res> {
  const result = await api.call<Req, Res>(method, request)
  if (!result.ok) throw new Error(result.error.message)
  return result.value
}
