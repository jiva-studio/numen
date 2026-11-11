/**
 * The registration of one reader or tab kind of a plugin: the tab kind the window holds for it, the
 * extensions it opens, and the undoing of both.
 */
import type {
  Disposable,
  FilePlacement,
  ReaderRegistration,
  Selector,
  TabKindRegistration,
  TextRange,
  Caught,
} from '@numen/plugin-sdk'
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
                { selector: Selector | null; ranges: readonly TextRange[] | null } | undefined
              if (!state) return
              // A new object each time, so a reader watching it moves to the same place again.
              if (selector) state.selector = { ...selector }
              if (ranges) state.ranges = [...ranges]
            })
            .catch((error: Caught) => reportError(deps, pluginId, error))
        },
        registration.icon,
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

function extractWhere(
  options?: FilePlacement | { readonly where?: FilePlacement | undefined },
): FilePlacement | undefined {
  return typeof options === 'string' ? options : options?.where
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
      (path, _title, options) => {
        const where = extractWhere(options)
        if (where === 'split' && deps.openTabAdjacent) {
          deps
            .openTabAdjacent(registration.kind, path)
            .catch((error: Caught) => reportError(deps, pluginId, error))
        } else {
          deps
            .openTab(registration.kind, path)
            .catch((error: Caught) => reportError(deps, pluginId, error))
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
