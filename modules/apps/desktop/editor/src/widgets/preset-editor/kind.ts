/**
 * Window registration and tab state for flashcard preset tabs.
 */
import { shallowRef } from 'vue'
import type { PlexShowing } from '@numen/ui'
import type { MessageWriter } from '@/shared/notices/messages'
import type { TabKind, WindowHandle } from '@/entities/tab/windowTabs'
import type { FileOpeners } from '@/entities/tab/openers'
import { fileOf, type PathRename } from '@/shared/paths'
import { PRESET } from '@/entities/tab/workspace'
import { NO_BOUNDS, type Presets, type SettingsBounds } from './types'
import { WORDS as words } from './words'
import type { PresetTabState, SettingValue } from './types'
import { createOpenPreset, createPresetState, readPreset, type OpenPreset } from './open'
import { flushWrites } from './flight'
import PresetTab from './ui/PresetTab.vue'

export type { PresetTabState, SettingValue }

/**
 * The presets one window has open.
 */
export function usePresetTab(
  core: Presets,
  handle: WindowHandle,
  puts: FileOpeners,
  said: MessageWriter,
  today: () => string,
) {
  const titles = new Map<string, string>()
  const bounds = shallowRef<SettingsBounds>(NO_BOUNDS)
  const open = new Map<string, OpenPreset>()

  const closePreset = (path: string) => {
    open.delete(path)
  }

  const holds = (id: string): PresetTabState | undefined => {
    const one = open.get(id)
    return one && createPresetState(one, id, handle, closePreset, core, bounds, said, today, titles)
  }

  const called = (path: string): string =>
    titles.get(path) || fileOf(path) || words.newPreset

  const kind: TabKind<PresetTabState, typeof PRESET> = {
    kind: PRESET,
    opens: (path) => {
      const one = createOpenPreset(path, today(), bounds.value)
      open.set(path, one)
      void readPreset(one, core, bounds, titles, today())
      return createPresetState(one, path, handle, closePreset, core, bounds, said, today, titles)
    },
    called: (one) => called(one.id),
    draws: PresetTab,
    identity: (path) => path,
    shuts: (one, id) => {
      one.shuts(id)
      return false
    },
    gone: () => {},
  }

  const shows = (path: string, title = '', showing: PlexShowing = 'here'): void => {
    if (title) titles.set(path, title)
    void (showing === 'beside' ? handle.beside(PRESET, path) : handle.opens(PRESET, path))
  }

  puts.holds('preset', shows)

  const changed = (paths: readonly string[], renamed: readonly PathRename[] = []): void => {
    for (const went of renamed) {
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
        flushWrites(one.flight, one.path.value, one.settings.value, core, said),
      ),
    )
  }

  return { kind, holds, changed, called, shows, flush }
}
