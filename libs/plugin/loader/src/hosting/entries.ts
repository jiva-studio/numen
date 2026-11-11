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
import { getReaderKindId } from './tabKinds'
import { createDeclaredTabKind, isDeclaredTabKind } from './pluginTabKind'
import type { PluginHostDeps } from './types'
import type { PluginItemRegistration, RegisteredReader, RegisteredTabKind } from './registry'

/** The entries that own the ids of the tabs a window holds: the tab kinds and the readers. */
export interface TabKindOwners {
  readonly readers: Map<string, RegisteredReader>
  readonly tabKinds: Map<string, RegisteredTabKind>
}


function requireFreeKind(
  owners: TabKindOwners,
  claim: { readonly pluginId: string; readonly what: 'reader' | 'tab kind'; readonly id: string },
): void {
  const { pluginId, what, id } = claim
  const kindId = what === 'reader' ? getReaderKindId(id) : id
  const sameId = (what === 'reader' ? owners.readers : owners.tabKinds).get(id)
  const sameKind =
    owners.tabKinds.get(kindId) ??
    [...owners.readers.values()].find((one) => getReaderKindId(one.reg.id) === kindId)
  const owner = sameId ?? sameKind
  if (!owner) return
  throw new Error(
    owner.pluginId === pluginId
      ? `Plugin "${pluginId}" registers the ${what} "${id}" twice`
      : `Plugin "${pluginId}" registers the ${what} "${id}", which plugin "${owner.pluginId}" already registers`,
  )
}

/**
 * Registers a reader. It throws where the window already holds the tab kind of the reader for a plugin:
 * the same tab kind, a reader of it, or the tab kind of that reader.
 */
export function registerReaderEntry(
  deps: PluginHostDeps,
  owners: TabKindOwners,
  item: PluginItemRegistration<ReaderRegistration>,
): Disposable {
  const { pluginId, registration, getContext } = item
  const { readers } = owners
  requireFreeKind(owners, { pluginId, what: 'reader', id: registration.id })
  const kindId = getReaderKindId(registration.id)
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
  owners: TabKindOwners,
  item: PluginItemRegistration<TabKindRegistration>,
): Disposable {
  const { pluginId, registration, getContext } = item
  const { tabKinds } = owners
  requireFreeKind(owners, { pluginId, what: 'tab kind', id: registration.kind })
  const kind = isDeclaredTabKind(registration)
    ? createDeclaredTabKind(pluginId, registration)
    : createCustomTabKind(pluginId, registration, { ...deps, getContext })
  const cleanups: (() => void)[] = [deps.registerKind(kind)]
  if (deps.registerEditor) {
    const unreg = deps.registerEditor(registration.kind, (path, _title, options) => {
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
    })
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
