/**
 * The registration of one reader or tab kind of a plugin: the tab kind the window holds for it, the
 * extensions it opens, and the undoing of both.
 */
import type { Component } from 'vue'
import type { Disposable, ReaderRegistration, TabKindRegistration } from '@numen/plugin-sdk'
import { reportError } from './events'
import { createReaderTabKind, createCustomTabKind } from './customTabKind'
import { createDeclaredTabKind, isDeclaredTabKind } from './pluginTabKind'
import type { PluginHostDeps } from './types'
import type { PluginItemRegistration, RegisteredReader, RegisteredTabKind } from './registry'

export function registerReaderEntry(
  deps: PluginHostDeps,
  readers: Map<string, RegisteredReader>,
  item: PluginItemRegistration<ReaderRegistration>,
): Disposable {
  const { pluginId, registration, getContext } = item
  const kindId = registration.id.startsWith('reader:')
    ? registration.id
    : `reader:${registration.id}`
  const cleanups: (() => void)[] = [
    deps.registerKind(createReaderTabKind(pluginId, registration, getContext)),
  ]
  for (const ext of deps.getReaderExtensions?.(pluginId, registration.id) ?? []) {
    cleanups.push(
      deps.registerExtensionReader(
        ext,
        (path, selector, ranges) => {
          deps
            .openTab(kindId, path)
            .then((tabId) => {
              if ((!selector && !ranges) || !deps.getTabState) return
              const state = deps.getTabState(tabId) as
                { selector?: unknown; ranges?: unknown } | undefined
              if (!state) return
              // A new object each time, so a reader watching it moves to the same place again.
              if (selector) state.selector = { ...selector }
              if (ranges) state.ranges = [...ranges]
            })
            .catch((error: unknown) => reportError(deps, pluginId, error))
        },
        registration.icon as Component | undefined,
      ),
    )
  }
  let isDisposed = false
  const entry: RegisteredReader = {
    pluginId,
    reg: registration,
    cleanup: () => {
      if (isDisposed) return
      isDisposed = true
      for (const fn of cleanups) fn()
      if (readers.get(registration.id) === entry) readers.delete(registration.id)
    },
  }
  readers.set(registration.id, entry)
  return { dispose: entry.cleanup }
}

function extractWhere(options?: unknown): unknown {
  if (typeof options === 'string') return options
  if (typeof options === 'object' && options !== null && 'where' in options) {
    return (options as { where?: unknown }).where
  }
  return undefined
}

export function registerTabKindEntry(
  deps: PluginHostDeps,
  tabKinds: Map<string, RegisteredTabKind>,
  item: PluginItemRegistration<TabKindRegistration>,
): Disposable {
  const { pluginId, registration, getContext } = item
  const kind = isDeclaredTabKind(registration)
    ? createDeclaredTabKind(pluginId, registration)
    : createCustomTabKind(pluginId, registration, { ...deps, getContext })
  const cleanups: (() => void)[] = [deps.registerKind(kind)]
  if (deps.registerEditor) {
    const unreg = deps.registerEditor(
      registration.kind,
      (path: string, _title?: string, options?: unknown) => {
        const where = extractWhere(options)
        if (where === 'split' && deps.openTabAdjacent) {
          deps
            .openTabAdjacent(registration.kind, path)
            .catch((error: unknown) => reportError(deps, pluginId, error))
        } else {
          deps
            .openTab(registration.kind, path)
            .catch((error: unknown) => reportError(deps, pluginId, error))
        }
      },
    )
    if (typeof unreg === 'function') cleanups.push(unreg)
  }
  let isDisposed = false
  const entry: RegisteredTabKind = {
    pluginId,
    reg: registration,
    cleanup: () => {
      if (isDisposed) return
      isDisposed = true
      for (const fn of cleanups) fn()
      if (tabKinds.get(registration.kind) === entry) tabKinds.delete(registration.kind)
    },
  }
  tabKinds.set(registration.kind, entry)
  return { dispose: entry.cleanup }
}
