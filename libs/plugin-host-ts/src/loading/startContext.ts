/**
 * The context one start of a plugin receives. Once the start is cancelled, the context registers
 * nothing more: each register and on call answers a disposable that holds nothing.
 */
import type { Disposable, PluginContext } from '@numen/plugin-sdk'

const isRegistration = (key: string | symbol): boolean =>
  typeof key === 'string' && /^(register|on)[A-Z]/.test(key)

const EMPTY: Disposable = { dispose: () => undefined }

/** The context of a start, which stops registering once `isCancelled` answers true. */
export function createStartContext(
  context: PluginContext,
  isCancelled: () => boolean,
): PluginContext {
  return new Proxy(context, {
    get(target, key, receiver) {
      const value: unknown = Reflect.get(target, key, receiver)
      if (!isRegistration(key) || typeof value !== 'function') return value
      return (...args: unknown[]): unknown =>
        isCancelled() ? EMPTY : (value as (...a: unknown[]) => unknown).apply(target, args)
    },
  })
}
