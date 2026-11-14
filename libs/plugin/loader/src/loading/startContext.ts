/**
 * The plugin context and the host services one start of a plugin receives. The start keeps every
 * disposable it is handed through them (`ui.commands.watch`, `vault.buffers.track`,
 * `ui.mountView`, a `register` call) and every one it pushes into its subscriptions, and
 * cancelling the start disposes them all. Once the
 * start is cancelled, a register or on call answers a disposable that holds nothing, and whatever
 * else the start is handed or pushes is disposed at once. Once the start has ended, both pass
 * everything through and keep nothing.
 */
import type { Disposable } from '@numen/plugin-sdk'
import type { PluginParts } from '../host'

const isRegistration = (group: string | symbol, key: string | symbol): boolean =>
  group === 'register' || (typeof key === 'string' && /^on[A-Z]/.test(key))

const EMPTY: Disposable = { dispose: () => undefined }

/** Whatever a member of a service holds: the proxies pass members on without knowing their type. */
type Member = object | string | number | bigint | boolean | symbol | null | undefined

const isDisposable = (value: Member): value is Disposable =>
  typeof value === 'object' &&
  value !== null &&
  typeof (value as Partial<Disposable>).dispose === 'function'

const isReplaceable = (target: object, key: string | symbol): boolean => {
  const descriptor = Reflect.getOwnPropertyDescriptor(target, key)
  return descriptor === undefined || descriptor.configurable || descriptor.writable === true
}

type Call = (...args: never[]) => Member

/** What the start keeps, and how it treats what it is handed. */
interface Keeper {
  readonly isCancelled: () => boolean
  readonly keep: (value: Member) => void
}

const wrapCall =
  (call: Call, owner: object, keeper: Keeper): Call =>
  (...args) => {
    const value = call.apply(owner, args)
    keeper.keep(value)
    return value
  }

function wrapService(service: object, keeper: Keeper): object {
  return new Proxy(service, {
    get(target, key) {
      const value: Member = Reflect.get(target, key)
      if (typeof value !== 'function' || !isReplaceable(target, key)) return value
      return wrapCall(value as Call, target, keeper)
    },
  })
}

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

/** The parts of a plugin as one start hands them over, and the end of that start either way. */
export interface StartScope {
  readonly wrapParts: (parts: PluginParts) => PluginParts
  readonly isCancelled: () => boolean
  readonly cancel: () => void
  readonly end: () => void
}

function createPartsWrapper(
  keeper: Keeper,
  subscriptionLists: Set<Disposable[]>,
): (parts: PluginParts) => PluginParts {
  const services = new WeakMap<object, object>()
  const getService = (value: object): object => {
    const service = services.get(value) ?? wrapService(value, keeper)
    services.set(value, service)
    return service
  }

  const wrapGroup = (name: string | symbol, group: object): object =>
    new Proxy(group, {
      get(target, key) {
        const value: Member = Reflect.get(target, key)
        if (!isReplaceable(target, key)) return value
        if (key === 'subscriptions' && Array.isArray(value)) return wrapSubscriptions(value, keeper)
        if (typeof value === 'function') {
          const call = wrapCall(value as Call, target, keeper)
          if (!isRegistration(name, key)) return call
          return (...args: never[]): Member => (keeper.isCancelled() ? EMPTY : call(...args))
        }
        if (typeof value !== 'object' || value === null || Array.isArray(value)) return value
        return getService(value)
      },
    })

  const groups = new WeakMap<object, object>()
  const wrapPart = <Part extends object>(part: Part): Part =>
    new Proxy(part, {
      get(target, key) {
        const value: Member = Reflect.get(target, key)
        if (typeof value !== 'object' || value === null || !isReplaceable(target, key)) return value
        const group = groups.get(value) ?? wrapGroup(key, value)
        groups.set(value, group)
        return group
      },
    })

  return (parts) => {
    if (Array.isArray(parts.plugin.identity.subscriptions)) {
      subscriptionLists.add(parts.plugin.identity.subscriptions)
    }
    return { plugin: wrapPart(parts.plugin), host: wrapPart(parts.host) }
  }
}

/** The scope of one start, which stops registering once `cancel` has run. */
export function createStartScope(): StartScope {
  let isEnded = false
  let isCancelled = false
  const startDisposables = new Set<Disposable>()
  const subscriptionLists = new Set<Disposable[]>()
  const keeper: Keeper = {
    isCancelled: () => !isEnded && isCancelled,
    keep: (value) => {
      if (isEnded || !isDisposable(value)) return
      if (keeper.isCancelled()) value.dispose()
      else startDisposables.add(value)
    },
  }
  const wrapParts = createPartsWrapper(keeper, subscriptionLists)

  return {
    wrapParts,
    isCancelled: keeper.isCancelled,
    cancel: () => {
      isCancelled = true
      for (const one of startDisposables) one.dispose()
      // A pushed disposable is disposed here, so the host does not dispose it a second time.
      for (const list of subscriptionLists) {
        const rest = list.filter((one) => !startDisposables.has(one))
        list.splice(0, list.length, ...rest)
      }
      startDisposables.clear()
    },
    end: () => {
      isEnded = true
      startDisposables.clear()
      subscriptionLists.clear()
    },
  }
}
