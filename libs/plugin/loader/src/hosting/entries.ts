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
import type { EditorPlacement, HostTabs, PluginHostDeps } from './types'

/** What a registration of a reader or a tab kind reads of the window. */
type EntryDeps = Pick<PluginHostDeps, 'window' | 'kinds' | 'tabs' | 'vault'>

const NO_TABS: HostTabs = {}
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

function undoSteps(steps: readonly (() => void)[]): void {
  for (const undo of [...steps].reverse()) undo()
}

function registerReaderExtensions(
  deps: EntryDeps,
  item: PluginItemRegistration<ReaderRegistration>,
  cleanups: (() => void)[],
): void {
  const { pluginId, registration } = item
  const { registerExtensionReader, getReaderExtensions } = deps.kinds
  const { tabs = NO_TABS } = deps
  const { openTab, getTabState } = tabs
  if (!registerExtensionReader || !openTab) return
  const kindId = getReaderKindId(registration.id)
  for (const ext of getReaderExtensions?.(pluginId, registration.id) ?? []) {
    cleanups.push(
      registerExtensionReader(
        ext,
        (path, selector, ranges) => {
          openTab(kindId, path)
            .then((tabId) => {
              if ((!selector && !ranges) || !getTabState) return
              const state = getTabState(tabId) as
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
}

/**
 * Registers a reader. It throws where the window already holds the tab kind of the reader for a plugin:
 * the same tab kind, a reader of it, or the tab kind of that reader.
 */
export function registerReaderEntry(
  deps: EntryDeps,
  owners: TabKindOwners,
  item: PluginItemRegistration<ReaderRegistration>,
): Disposable {
  const { pluginId, registration, getContext } = item
  const { readers } = owners
  requireFreeKind(owners, { pluginId, what: 'reader', id: registration.id })
  const cleanups: (() => void)[] = []
  try {
    cleanups.push(deps.kinds.registerKind(createReaderTabKind(pluginId, registration, getContext)))
    registerReaderExtensions(deps, item, cleanups)
  } catch (error) {
    undoSteps(cleanups)
    throw error
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

type EditorOpenOptions = FilePlacement | EditorPlacement | null | undefined

function extractWhere(options?: EditorOpenOptions): FilePlacement | undefined {
  return typeof options === 'string' ? options : options?.where
}

function extractLine(options?: EditorOpenOptions): number | undefined {
  return typeof options === 'object' ? options?.line : undefined
}

export function registerTabKindEntry(
  deps: EntryDeps,
  owners: TabKindOwners,
  item: PluginItemRegistration<TabKindRegistration> & { readonly isNoteEditor: boolean },
): Disposable {
  const { pluginId, registration, getContext, isNoteEditor } = item
  const { tabKinds } = owners
  requireFreeKind(owners, { pluginId, what: 'tab kind', id: registration.kind })
  const kind = isDeclaredTabKind(registration)
    ? createDeclaredTabKind(pluginId, registration)
    : createCustomTabKind(pluginId, registration, { ...deps, getContext, isNoteEditor })
  const cleanups: (() => void)[] = []
  const { registerEditor } = deps.kinds
  const { tabs = NO_TABS } = deps
  const { openTab, openTabAdjacent, getTabState } = tabs
  try {
    cleanups.push(deps.kinds.registerKind(kind))
    if (registerEditor && openTab) {
      const unreg = registerEditor(registration.kind, (path, _title, options) => {
        const where = extractWhere(options)
        const line = extractLine(options)
        const opening =
          where === 'split' && openTabAdjacent
            ? openTabAdjacent(registration.kind, path)
            : openTab(registration.kind, path)
        opening
          .then((tabId) => {
            if (line === undefined) return
            const state = getTabState?.(tabId) as
              { revealLine?: (line: number) => void } | undefined
            state?.revealLine?.(line)
          })
          .catch((error: Caught) => reportError(deps, pluginId, error))
      })
      if (typeof unreg === 'function') cleanups.push(unreg)
    }
  } catch (error) {
    undoSteps(cleanups)
    throw error
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
