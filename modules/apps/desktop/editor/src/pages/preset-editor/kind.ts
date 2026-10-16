/**
 * Window registration and tab state for flashcard preset tabs.
 */
import { shallowRef, type Ref } from 'vue'
import type { PlexDestination } from '@numen/ui'
import type { MessageWriter } from '@/shared/notices/messages'
import type { FileOpeners, OpenPlacement, TabKind, WindowHandle } from '@/entities/tab'
import { getFileName, type PathRename } from '@/shared/paths'
import { PRESET } from '@/entities/tab'
import { NO_BOUNDS, type Presets, type SettingsBounds } from './types'
import { WORDS as words } from './words'
import type { PresetTabState, SettingValue } from './types'
import {
  createOpenPreset,
  createPresetState,
  readPreset,
  type OpenPreset,
  type PresetStateContext,
} from './model/open'
import { flushWrites } from './model/writeQueue'
import PresetTab from './ui/PresetTab.vue'

export type { PresetTabState, SettingValue }

export interface PresetTabDeps {
  core: Presets
  handle: WindowHandle
  tabOpeners: FileOpeners
  writeMessage: MessageWriter
  today: () => string
}

function applyPathRenames(
  open: Map<string, OpenPreset>,
  titles: Map<string, string>,
  renames: readonly PathRename[],
): void {
  for (const went of renames) {
    const title = titles.get(went.from)
    // Stryker disable next-line ConditionalExpression: Missing title returns undefined on get either way.
    if (title !== undefined) titles.set(went.to, title)
    titles.delete(went.from)
    const one = open.get(went.from)
    if (!one) continue
    open.delete(went.from)
    one.path.value = went.to
    open.set(went.to, one)
  }
}

interface KindDeps {
  handle: WindowHandle
  context: PresetStateContext
  open: Map<string, OpenPreset>
  titles: Map<string, string>
  bounds: Ref<SettingsBounds>
  core: Presets
  today: () => string
}

function createPresetKind(deps: KindDeps): TabKind<PresetTabState, typeof PRESET> {
  const { context, open, titles, bounds, core, today } = deps
  const getTitle = (path: string): string =>
    titles.get(path) || getFileName(path) || words.newPreset

  return {
    kind: PRESET,
    open: (path) => {
      const one = createOpenPreset(path, today(), bounds.value)
      open.set(path, one)
      void readPreset(one, { core, bounds, titles, today: today() })
      return createPresetState(one, path, context)
    },
    getTitle: (one) => getTitle(one.id),
    pane: PresetTab,
    identity: (path) => path,
    getTarget: (one) => ({ path: one.id, title: getTitle(one.id) }),
    getOpenTab: (one) => ({ path: one.id }),
    onClose: (one, id) => {
      one.close(id)
      return false
    },
    onDestroy: () => {},
  }
}

/**
 * The presets one window has open.
 */
export function usePresetTab(deps: PresetTabDeps) {
  const { core, handle, tabOpeners, writeMessage, today } = deps
  const titles = new Map<string, string>()
  const bounds = shallowRef<SettingsBounds>(NO_BOUNDS)
  const open = new Map<string, OpenPreset>()

  const closePreset = (path: string) => {
    open.delete(path)
  }

  const context: PresetStateContext = {
    handle,
    onClosed: closePreset,
    core,
    bounds,
    writeMessage,
    today,
    titles,
  }

  const getState = (id: string): PresetTabState | undefined => {
    const one = open.get(id)
    return one ? createPresetState(one, id, context) : undefined
  }

  const getTitle = (path: string): string =>
    titles.get(path) || getFileName(path) || words.newPreset

  const readOne = (one: OpenPreset) => readPreset(one, { core, bounds, titles, today: today() })

  const kind = createPresetKind({ handle, context, open, titles, bounds, core, today })

  const openPreset = (
    path: string,
    title = '',
    // Stryker disable next-line StringLiteral: Default placement opens in the active tab pane.
    how: PlexDestination | OpenPlacement = 'here',
  ): void => {
    if (title) titles.set(path, title)
    const where = typeof how === 'string' ? how : (how?.where ?? 'here')
    void (where === 'beside' ? handle.openTabBeside(PRESET, path) : handle.openTab(PRESET, path))
  }

  tabOpeners.registerEditor?.('preset', openPreset)

  const applyPathChanges = (
    paths: readonly string[],
    // Stryker disable next-line ArrayDeclaration: an entry that is not a rename names no file, and every walk over renames skips it.
    renames: readonly PathRename[] = [],
  ): void => {
    applyPathRenames(open, titles, renames)
    for (const path of paths) {
      const one = open.get(path)
      if (one) void readOne(one)
    }
  }

  const flush = async (): Promise<void> => {
    await Promise.all(
      [...open.values()].map((one) =>
        flushWrites(one.writeQueue, {
          path: one.path.value,
          settings: one.settings.value,
          core,
          writeMessage,
        }),
      ),
    )
  }

  return { kind, getState, applyPathChanges, getTitle, openPreset, flush }
}
