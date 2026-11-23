import type { Disposable, Subscriptions } from '@numen/plugin-sdk'

/** What a plugin context keeps for disposal: the view the plugin gets, and the list only the host reaches. */
export interface SubscriptionList {
  readonly view: Subscriptions
  add(disposable: Disposable): void
  remove(disposable: Disposable): void
  /** Disposes what a plugin subscribed, the last added first, including what the disposals add. */
  disposeAll(): void
}

/** A list whose view lets a plugin add and count but never remove, replace or clear. */
export function createSubscriptionList(): SubscriptionList {
  const items: Disposable[] = []
  const view: Subscriptions = Object.freeze({
    push: (...added: Disposable[]) => items.push(...added),
    get length() {
      return items.length
    },
  })
  return {
    view,
    add: (disposable) => {
      items.push(disposable)
    },
    remove: (disposable) => {
      const index = items.indexOf(disposable)
      if (index !== -1) items.splice(index, 1)
    },
    disposeAll: () => {
      while (items.length > 0) items.pop()?.dispose()
    },
  }
}
