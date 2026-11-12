/**
 * The calls on models of a plugin, carried out over ModelService for the rows of that plugin.
 */
import type { Caught, ModelCallFailure, ModelClient } from '@numen/core-client'
import type { ChatEvent, ModelCallOptions, ModelError, ModelHost, Result } from '@numen/plugin-sdk'

const readError = (failure: ModelCallFailure): ModelError => ({
  error: failure.errorCode,
  message: failure.detail,
  failures: failure.failures.map((one) => ({
    connectionName: one.connectionName,
    reason: one.reason,
  })),
})

const readResult = <T>(
  answer: { ok: true; value: T } | { ok: false; error: ModelCallFailure },
): Result<T, ModelError> =>
  answer.ok ? { ok: true, value: answer.value } : { ok: false, error: readError(answer.error) }

const readCancellation = (error: Caught, signal: AbortSignal | undefined): ModelError => {
  if (signal?.aborted !== true) throw error
  return {
    error: 'canceled',
    message: error instanceof Error ? error.message : String(error),
    failures: [],
  }
}

const settle = async <T>(
  signal: AbortSignal | undefined,
  call: () => Promise<{ ok: true; value: T } | { ok: false; error: ModelCallFailure }>,
): Promise<Result<T, ModelError>> => {
  try {
    return readResult(await call())
  } catch (error) {
    return { ok: false, error: readCancellation(error, signal) }
  }
}

async function* streamChat(
  open: () => ReturnType<ModelClient['chat']>,
  options: ModelCallOptions | undefined,
): AsyncGenerator<ChatEvent, ModelError | null> {
  try {
    const end = yield* open()
    return end === null ? null : readError(end)
  } catch (error) {
    return readCancellation(error, options?.signal)
  }
}

/** The model calls a plugin makes, with the plugin id the loader gives it. */
export const createPluginModels = (client: ModelClient, pluginId: string): ModelHost => ({
  complete: (request, options) =>
    settle(options?.signal, () => client.complete(pluginId, request, options)),
  embed: (request, options) =>
    settle(options?.signal, () => client.embed(pluginId, request.texts, options)),
  getEmbeddingProfile: (options) =>
    settle(options?.signal, () => client.getEmbeddingProfile(pluginId, options)),
  chat: (request, options) => streamChat(() => client.chat(pluginId, request, options), options),
})
