import { createClient } from '@connectrpc/connect'
import { HostDependencyService } from '@numen/protocol'
import { formatErrorMessage } from './errors'
import { asFailure, asValue, type Result } from './result'
import { transport } from './transport'

const dependencies = createClient(HostDependencyService, transport)

/** Requests the local path of a model file from the core's shared store. */
export async function fetchModelFile(
  url: string,
  sha256?: string,
  signal?: AbortSignal,
): Promise<Result<string>> {
  try {
    const answer = await dependencies.fetchModelFile({ url, sha256: sha256 ?? '' }, { signal })
    return answer.error ? asFailure(answer.error) : asValue(answer.path)
  } catch (error) {
    return asFailure(formatErrorMessage(error))
  }
}
