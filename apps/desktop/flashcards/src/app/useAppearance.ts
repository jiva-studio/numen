/**
 * The appearance commands as the window's command palette: opened with Ctrl+P (Cmd+P), filled with
 * the themes and modes the application offers, and kept in step with the themes it reports.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { formatErrorMessage } from '@numen/core-client'
import { createFollower } from '@numen/ui'
import type { NoticeQueue } from '@numen/workbench'
import { pluginService } from '@/shared/clients'
import { createAppearance, type ThemeClient } from './appearance'
import { themeClient } from './themeClient'
import { WINDOW_SOURCE } from './useWindowTasks'

/** What the commands are given in place of the window's own services. */
export interface AppearanceServices {
  readonly client?: ThemeClient
  readonly listPluginNames?: () => Promise<ReadonlyMap<string, string>>
}

const listInstalledPluginNames = async (): Promise<ReadonlyMap<string, string>> =>
  new Map((await pluginService.getPlugins({})).plugins.map((one) => [one.id, one.name]))

const isToggleChord = (event: KeyboardEvent): boolean =>
  !event.defaultPrevented &&
  (event.ctrlKey || event.metaKey) &&
  !event.altKey &&
  !event.shiftKey &&
  event.key.toLowerCase() === 'p'

export function useAppearance(notices: NoticeQueue, services: AppearanceServices = {}) {
  const client = services.client ?? themeClient
  const listPluginNames = services.listPluginNames ?? listInstalledPluginNames
  const pluginNames = ref<ReadonlyMap<string, string>>(new Map())
  const showError = (text: string) => notices.show(WINDOW_SOURCE, text, 'error')
  const appearance = createAppearance({
    client,
    getPluginName: (pluginId) => pluginNames.value.get(pluginId) ?? pluginId,
    onError: showError,
  })

  const isOpen = ref(false)
  const query = ref('')
  const groups = computed(() => appearance.getGroups(query.value))

  /** The plugin names read again, so a group is titled with the name the host holds now. */
  const refreshPluginNames = async () => {
    try {
      pluginNames.value = await listPluginNames()
    } catch (error) {
      showError(formatErrorMessage(error))
    }
  }

  let isMounted = true
  const abortController = new AbortController()
  const follow = createFollower({
    isOpen: () => isMounted,
    setLost: showError,
    wait: (ms) => new Promise((then) => setTimeout(then, ms)),
  })

  /* -------------------------------- Handlers -------------------------------- */
  function onKeyDown(event: KeyboardEvent) {
    if (!isToggleChord(event)) return
    event.preventDefault()
    if (isOpen.value) {
      onDismiss()
      return
    }
    query.value = ''
    isOpen.value = true
    // Stryker disable next-line all: refreshPluginNames and appearance.refresh catch their own errors
    Promise.all([refreshPluginNames(), appearance.refresh()]).catch((error: unknown) =>
      showError(formatErrorMessage(error)),
    )
  }

  async function onChoose(item: string) {
    isOpen.value = false
    await appearance.chooseItem(item)
  }

  function onDismiss() {
    isOpen.value = false
  }

  /* ---------------------------------- Hooks --------------------------------- */
  onMounted(() => {
    window.addEventListener('keydown', onKeyDown)
    // Stryker disable next-line all: appearance.refresh catches its own errors
    appearance.refresh().catch((error: unknown) => showError(formatErrorMessage(error)))
    follow(
      () => client.watchThemes(abortController.signal),
      async (names) => {
        if (names.length > 0) await appearance.refresh()
      },
    ).catch(
      // Stryker disable next-line all: createFollower catches stream errors and reports them through setLost
      (error: unknown) => showError(formatErrorMessage(error)),
    )
  })
  onUnmounted(() => {
    isMounted = false
    abortController.abort()
    window.removeEventListener('keydown', onKeyDown)
  })

  return { isOpen, query, groups, onChoose, onDismiss }
}
