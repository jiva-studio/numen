/**
 * The assembled window: the plugins of its target, loaded over HTTP and activated; the vaults the
 * user chooses from; and the tab the window opens on the chosen vault.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import {
  INTERFACE_WAIT_MS,
  createPluginLoader,
  createWindowHost,
  usePluginStyles,
  type PluginLoaderDeps,
} from '@numen/plugin-loader'
import { useWindowFrame, type WindowFrame } from '@numen/workbench'
import { runInBackground } from '@numen/ui'
import { createWindowAgent } from '@/entities/agent'
import { listPlugins } from '@/entities/plugin'
import { useVaults } from '@/entities/vault'
import { useAppearance } from '@/features/appearance'
import { useExternalLinks } from '@/features/external-links'
import { useWindowTasks } from '@/features/window-tasks'
import { formatError } from '@/shared/api/errors'
import { WINDOW_SOURCE } from '@/shared/config/window'
import { createShellHost, createShellServices, type ShellServices } from '../plugins/host'
import { createWindowCommands } from '../plugins/windowCommands'
import { createWindowKeys } from '../plugins/windowKeys'
import { createWindowVaults } from '../plugins/windowVaults'

/** What the window is given: how a plugin module is fetched and imported. */
export interface WindowDeps {
  readonly fetch?: PluginLoaderDeps['fetch']
  readonly dynamicImport?: PluginLoaderDeps['dynamicImport']
}

function createServices(
  window: Pick<ReturnType<typeof createShellHost>, 'tab' | 'openTab' | 'notices'> & {
    readonly vaults: ReturnType<typeof useVaults>['vaults']
    readonly frame: WindowFrame
  },
): ShellServices {
  const { tab, notices, frame } = window
  return {
    commands: createWindowCommands(),
    vaults: createWindowVaults({
      list: window.vaults,
      getTabVault: () => tab.value?.vault ?? null,
      openTab: window.openTab,
    }),
    window: createWindowHost(frame, (error) =>
      notices.show(WINDOW_SOURCE, formatError(error), 'error'),
    ),
  }
}

export function createWindow(deps: WindowDeps = {}) {
  const windowAgent = createWindowAgent()
  const services = createShellServices()
  const frame = useWindowFrame()
  const { host, front, tab, openTab, notices, mountView, titleBarView } = createShellHost(
    windowAgent.agent,
    services.get,
    () => frame.platform.value,
  )
  const loader = createPluginLoader({
    host,
    ...(deps.fetch ? { fetch: deps.fetch } : {}),
    ...(deps.dynamicImport ? { dynamicImport: deps.dynamicImport } : {}),
  })
  const isLoading = ref(true)
  const listError = ref('')
  const { vaults, isListing } = useVaults((text) => notices.show(WINDOW_SOURCE, text, 'error'))
  services.provide(createServices({ vaults, tab, openTab, notices, frame }))
  const onKeyDown = createWindowKeys({
    getTab: () => tab.value,
    getVaults: () => (front.value ? vaults.value : []),
    openVault: openTab,
  })
  useWindowTasks(notices)
  useExternalLinks(notices)
  const appearance = useAppearance(notices)

  /**
   * True once the plugins of the window slots have started or INTERFACE_WAIT_MS has passed since the
   * window mounted: the window is drawn then.
   */
  const isStarted = ref(false)
  let startTimer: ReturnType<typeof setTimeout> | undefined
  const onStarted = (): void => {
    isStarted.value = true
    clearTimeout(startTimer)
  }

  onMounted(async () => {
    window.addEventListener('keydown', onKeyDown)
    startTimer = setTimeout(onStarted, INTERFACE_WAIT_MS)
    runInBackground(loader.waitForInterface().then(onStarted), () => undefined)
    try {
      await loader.trackPlugins((await listPlugins()).map((one) => one.id))
      await loader.start()
    } catch (error) {
      listError.value = formatError(error)
    } finally {
      isLoading.value = false
      onStarted()
    }
  })
  onUnmounted(() => {
    clearTimeout(startTimer)
    window.removeEventListener('keydown', onKeyDown)
    loader.close().catch((error: unknown) => {
      listError.value = formatError(error)
    })
    windowAgent.dispose()
  })

  /** Why the window draws no view: the plugins could not be listed, or each one failed to start. */
  const failures = computed<readonly string[]>(() => [
    ...(listError.value ? [listError.value] : []),
    ...loader.getFailedPlugins().map((one) => `${one.id}: ${one.error}`),
  ])

  usePluginStyles(() => (front.value ? [front.value] : []))

  return {
    front,
    tab,
    openTab,
    vaults: { list: vaults, isListing },
    plugins: { failures, isLoading, isStarted, titleBarView },
    notices,
    mountView,
    appearance,
  }
}
