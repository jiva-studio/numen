/**
 * The tabs by kind: which kinds a window can open, the tabs of a plugin whose kind goes away, and
 * what a kind may ask of the window.
 */
import { computed } from 'vue'
import type { Result } from '@/shared/lib/result'
import type { AnyTabKind } from '../lib/kinds'
import {
  createPlaceholders,
  createTabNavigation,
  createTabOpeners,
  createWindowHandle,
} from './tabOperations'
import type { TabStore } from './tabStore'

export function useTabGroups(store: TabStore) {
  const { byKind } = store
  const placeholders = createPlaceholders(store)
  const navigation = createTabNavigation(store, createTabOpeners(store))
  const kinds = computed(() => [...byKind.value.values()])

  const requireFree = (registered: ReadonlyMap<string, AnyTabKind>, kind: AnyTabKind): void => {
    const holder = registered.get(kind.kind)
    if (!holder) return
    throw new Error(
      `The tab kind "${kind.kind}" is registered already${holder.pluginId ? ` by plugin "${holder.pluginId}"` : ''}`,
    )
  }
  const registerKinds = (declared: readonly AnyTabKind[]) => {
    const registered = new Map(byKind.value)
    for (const one of declared) {
      requireFree(registered, one)
      registered.set(one.kind, one)
    }
    byKind.value = registered
  }
  const registerKind = (kind: AnyTabKind): (() => void) => {
    requireFree(byKind.value, kind)
    byKind.value = new Map(byKind.value).set(kind.kind, kind)
    placeholders.reopenTabsOf(kind)
    return () => {
      if (byKind.value.get(kind.kind) !== kind) return
      const registered = new Map(byKind.value)
      registered.delete(kind.kind)
      byKind.value = registered
    }
  }
  const hasKind = (kind: string): boolean => byKind.value.has(kind)
  /** The first kind registered that answers yes, and nothing where none does. */
  const findKind = (isWanted: (kind: AnyTabKind) => boolean): AnyTabKind | undefined =>
    kinds.value.find(isWanted)
  /** Creates a file with the named kind, which edits it; undefined when no kind has that name. */
  const createFileOfKind = (
    kind: string,
    folder: string,
    name: string,
  ): Promise<Result<{ path: string }, string>> | undefined =>
    byKind.value.get(kind)?.address?.createFile?.(folder, name)

  return {
    registerKinds,
    registerKind,
    hasKind,
    findKind,
    closeTabsOfPlugin: placeholders.closeTabsOfPlugin,
    createFileOfKind,
    handle: createWindowHandle(store),
    sendText: navigation.sendText,
  }
}
