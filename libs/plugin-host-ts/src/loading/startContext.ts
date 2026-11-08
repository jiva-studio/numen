/**
 * The context one start of a plugin receives. The start keeps every disposable it is handed through
 * the context and its services (`commands.watch`, `buffers.track`, `mountView`, a register call) and
 * every one it pushes into its subscriptions, and cancelling the start disposes them all. Once the
 * start is cancelled, a register or on call answers a disposable that holds nothing, and whatever
 * else the start is handed or pushes is disposed at once. Once the start has ended, the context
 * passes everything through and keeps nothing.
 */
import type { Disposable, PluginContext } from '@numen/plugin-sdk'

const isRegistration = (key: string | symbol): boolean =>
  typeof key === 'string' && /^(register|on)[A-Z]/.test(key)

const EMPTY: Disposable = { dispose: () => undefined }

const isDisposable = (value: unknown): value is Disposable =>
  typeof value === 'object' &&
  value !== null &&
  typeof (value as Partial<Disposable>).dispose === 'function'

/** Whether a proxy may answer another value for the property: it is neither frozen nor fixed. */
const isReplaceable = (target: object, key: string | symbol): boolean => {
  const descriptor = Reflect.getOwnPropertyDescriptor(target, key)
  return descriptor === undefined || descriptor.configurable || descriptor.writable === true
}

type Call = (...args: unknown[]) => unknown

/** What the start keeps, and how it treats what it is handed. */
interface Keeper {
  readonly isCancelled: () => boolean
  /** Keeps a disposable the start was handed, or disposes it once the start is cancelled. */
  readonly keep: (value: unknown) => void
}

/** A function of the context or of a service whose disposable results are kept. */
const wrapCall =
  (call: Call, owner: object, keeper: Keeper): Call =>
  (...args) => {
    const value = call.apply(owner, args)
    keeper.keep(value)
    return value
  }

/** A service of the context, whose functions hand their disposables to the keeper. */
function wrapService(service: object, keeper: Keeper): object {
  return new Proxy(service, {
    get(target, key) {
      const value: unknown = Reflect.get(target, key)
      if (typeof value !== 'function' || !isReplaceable(target, key)) return value
      return wrapCall(value as Call, target, keeper)
    },
  })
}

/** The subscriptions of the context, whose pushed disposables are kept. */
function wrapSubscriptions(subscriptions: Disposable[], keeper: Keeper): Disposable[] {
  return new Proxy(subscriptions, {
    get(target, key) {
      if (key !== 'push') return Reflect.get(target, key)
      return (...items: Disposable[]): number => {
        for (const one of items) keeper.keep(one)
        return keeper.isCancelled() ? target.length : target.push(...items)
      }
    },
  })
}

/** The context of a plugin as one start hands it over, and the end of that start either way. */
export interface StartScope {
  readonly wrapContext: (context: PluginContext) => PluginContext
  /** Whether `cancel` has run and the start has not ended before it. */
  readonly isCancelled: () => boolean
  /** Disposes what the start was handed and pushed; from then on each one is disposed as it comes. */
  readonly cancel: () => void
  /** Ends the start: what it registered stays with the plugin, and nothing is kept any more. */
  readonly end: () => void
}

/** The scope of one start, which stops registering once `cancel` has run. */
export function createStartScope(): StartScope {
  let isEnded = false
  let isCancelled = false
  const kept = new Set<Disposable>()
  const subscriptionLists = new Set<Disposable[]>()
  const keeper: Keeper = {
    isCancelled: () => !isEnded && isCancelled,
    keep: (value) => {
      if (isEnded || !isDisposable(value)) return
      if (keeper.isCancelled()) value.dispose()
      else kept.add(value)
    },
  }
  const services = new WeakMap<object, object>()
  const getService = (value: object): object => {
    const service = services.get(value) ?? wrapService(value, keeper)
    services.set(value, service)
    return service
  }

  const wrapContext = (context: PluginContext): PluginContext => {
    if (Array.isArray(context.subscriptions)) subscriptionLists.add(context.subscriptions)
    return new Proxy(context, {
      get(target, key) {
        const value: unknown = Reflect.get(target, key)
        if (!isReplaceable(target, key)) return value
        if (key === 'subscriptions' && Array.isArray(value)) return wrapSubscriptions(value, keeper)
        if (typeof value === 'function') {
          const call = wrapCall(value as Call, target, keeper)
          if (!isRegistration(key)) return call
          return (...args: unknown[]): unknown => (keeper.isCancelled() ? EMPTY : call(...args))
        }
        if (typeof value !== 'object' || value === null || Array.isArray(value)) return value
        return getService(value)
      },
    })
  }

  return {
    wrapContext,
    isCancelled: keeper.isCancelled,
    cancel: () => {
      isCancelled = true
      for (const one of kept) one.dispose()
      // A pushed disposable is disposed here, so the host does not dispose it a second time.
      for (const list of subscriptionLists) {
        const rest = list.filter((one) => !kept.has(one))
        list.splice(0, list.length, ...rest)
      }
      kept.clear()
    },
    end: () => {
      isEnded = true
      kept.clear()
      subscriptionLists.clear()
    },
  }
}
