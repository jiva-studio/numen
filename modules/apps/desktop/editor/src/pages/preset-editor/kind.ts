/**
 * Window registration and tab state for flashcard preset tabs.
 */
import { shallowRef } from 'vue'
import type { PlexDestination } from '@numen/ui'
import type { MessageWriter } from '@/shared/notices/messages'
import type { TabKind, WindowHandle } from '@/entities/tab'
import type { FileOpeners } from '@/entities/tab'
import { fileOf, type PathRename } from '@/shared/paths'
import { PRESET } from '@/entities/tab'
import { NO_BOUNDS, type Presets, type SettingsBounds } from './types'
import { WORDS as words } from './words'
import type { PresetTabState, SettingValue } from './types'
import { createOpenPreset, createPresetState, readPreset, type OpenPreset } from './model/open'
import { flushWrites } from './model/flight'
import PresetTab from './ui/PresetTab.vue'

export type { PresetTabState, SettingValue }

/**
 * The presets one window has open.
 */
export function usePresetTab(
  core: Presets,
  handle: WindowHandle,
  tabOpeners: FileOpeners,
  writeMessage: MessageWriter,
  today: () => string,
) {
  const titles = new Map<string, string>()
  const bounds = shallowRef<SettingsBounds>(NO_BOUNDS)
  const open = new Map<string, OpenPreset>()

  const closePreset = (path: string) => {
    open.delete(path)
  }

  const getState = (id: string): PresetTabState | undefined => {
    const one = open.get(id)
    return (
      one &&
      createPresetState(one, id, handle, closePreset, core, bounds, writeMessage, today, titles)
    )
  }

  const getTitle = (path: string): string => titles.get(path) || fileOf(path) || words.newPreset

  const kind: TabKind<PresetTabState, typeof PRESET> = {
    kind: PRESET,
    open: (path) => {
      const one = createOpenPreset(path, today(), bounds.value)
      open.set(path, one)
      void readPreset(one, core, bounds, titles, today())
      return createPresetState(
        one,
        path,
        handle,
        closePreset,
        core,
        bounds,
        writeMessage,
        today,
        titles,
      )
    },
    getTitle: (one) => getTitle(one.id),
    pane: PresetTab,
    identity: (path) => path,
    onClose: (one, id) => {
      one.close(id)
      return false
    },
    onDestroy: () => {},
  }

  const openPreset = (path: string, title = '', how: PlexDestination = 'here'): void => {
    if (title) titles.set(path, title)
    void (how === 'beside' ? handle.openTabBeside(PRESET, path) : handle.openTab(PRESET, path))
  }

  tabOpeners.registerEditor('preset', openPreset)

  const applyPathChanges = (
    paths: readonly string[],
    renames: readonly PathRename[] = [],
  ): void => {
    for (const went of renames) {
      const title = titles.get(went.from)
      if (title !== undefined) titles.set(went.to, title)
      titles.delete(went.from)
      const one = open.get(went.from)
      if (!one) continue
      open.delete(went.from)
      one.path.value = went.to
      open.set(went.to, one)
    }
    for (const path of paths) {
      const one = open.get(path)
      if (one) void readPreset(one, core, bounds, titles, today())
    }
  }

  const flush = async (): Promise<void> => {
    await Promise.all(
      [...open.values()].map((one) =>
        flushWrites(one.flight, one.path.value, one.settings.value, core, writeMessage),
      ),
    )
  }

  return { kind, getState, applyPathChanges, getTitle, openPreset, flush }
}
