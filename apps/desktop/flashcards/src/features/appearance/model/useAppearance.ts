/**
 * The appearance commands as the window's command palette: opened with Ctrl+P (Cmd+P), filled with
 * the themes and modes the application offers, and kept in step with the themes it reports.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { createFollower } from '@numen/ui'
import type { NoticeQueue } from '@numen/workbench'
import { listPlugins } from '@/entities/plugin'
import { formatError, type Caught } from '@/shared/api/errors'
import { WINDOW_SOURCE } from '@/shared/lib/window'
import { themeClient } from '../api/theme'
import { createAppearance } from './appearance'

const isToggleChord = (event: KeyboardEvent): boolean =>
  !event.defaultPrevented &&
  (event.ctrlKey || event.metaKey) &&
  !event.altKey &&
  !event.shiftKey &&
  event.key.toLowerCase() === 'p'

export function useAppearance(notices: NoticeQueue) {
  const pluginNames = ref<ReadonlyMap<string, string>>(new Map())
  const showError = (text: string) => notices.show(WINDOW_SOURCE, text, 'error')
  const appearance = createAppearance({
    client: themeClient,
    getPluginName: (pluginId) => pluginNames.value.get(pluginId) ?? pluginId,
    onError: showError,
  })

  const isOpen = ref(false)
  const query = ref('')
  const groups = computed(() => appearance.getGroups(query.value))

  /** The plugin names read again, so a group is titled with the name the host holds now. */
  const refreshPluginNames = async () => {
    try {
      pluginNames.value = new Map((await listPlugins()).map((one) => [one.id, one.name]))
    } catch (error) {
      showError(formatError(error))
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
    Promise.all([refreshPluginNames(), appearance.refresh()]).catch((error: Caught) =>
      showError(formatError(error)),
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
    globalThis.addEventListener('keydown', onKeyDown)
    // Stryker disable next-line all: appearance.refresh catches its own errors
    appearance.refresh().catch((error: Caught) => showError(formatError(error)))
    follow(
      () => themeClient.watchThemes(abortController.signal),
      async (names) => {
        if (names.length > 0) await appearance.refresh()
      },
    ).catch(
      // Stryker disable next-line all: createFollower catches stream errors and reports them through setLost
      (error: Caught) => showError(formatError(error)),
    )
  })
  onUnmounted(() => {
    isMounted = false
    abortController.abort()
    globalThis.removeEventListener('keydown', onKeyDown)
  })

  return { isOpen, query, groups, onChoose, onDismiss }
}
