/**
 * The appearance of the window and its palette: the theme and mode the settings name, applied to the
 * page and kept in step with the themes the application reports, and chosen from the palette.
 */
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { runInBackground } from '@numen/ui'
import { createWindowAppearance, useAppearancePalette, type NoticeQueue } from '@numen/workbench'
import type { PluginLister } from '@/entities/plugin'
import { formatError, type Caught } from '@/shared/api/errors'
import { WINDOW_SOURCE } from '@/shared/lib/window'
import type { ThemeClient } from '../api/theme'
import { WORDS } from '../words'

/** The calls the appearance reads the themes and the names of the plugins through. */
export interface AppearanceApi {
  readonly themes: ThemeClient
  readonly listPlugins: PluginLister
}

export function useAppearance(api: AppearanceApi, notices: NoticeQueue) {
  const { themes, listPlugins } = api
  const showError = (text: string) => notices.show(WINDOW_SOURCE, text, 'error')
  const pluginNames = ref<ReadonlyMap<string, string>>(new Map())
  const appearance = createWindowAppearance({ core: themes, words: WORDS, write: showError })
  watch(appearance.state.lost, (text) => {
    if (text !== '') showError(text)
  })

  /** The plugin names read again, so a group is titled with the name the host holds now. */
  const refreshPluginNames = async () => {
    try {
      pluginNames.value = new Map((await listPlugins()).map((one) => [one.id, one.name]))
    } catch (error) {
      showError(formatError(error))
    }
  }

  const palette = useAppearancePalette({
    appearance,
    words: WORDS,
    getPluginName: (pluginId) => pluginNames.value.get(pluginId) ?? pluginId,
    writeChoice: (theme, mode) =>
      themes.writeAppearance(theme, mode).catch((error: Caught) => formatError(error)),
    onOpen: async () => {
      await Promise.all([refreshPluginNames(), appearance.reload()])
    },
    write: showError,
  })

  onMounted(() => {
    runInBackground(appearance.start(), (error: Caught) => showError(formatError(error)))
  })
  onUnmounted(() => appearance.close())

  return palette
}
