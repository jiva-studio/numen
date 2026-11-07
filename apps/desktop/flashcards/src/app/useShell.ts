/**
 * The assembled shell: the plugins of the window's target, loaded over HTTP and
 * activated; the vaults the user chooses from; and the tab the window opens on
 * the chosen vault.
 */
import { computed, onMounted, onUnmounted, ref, shallowRef } from 'vue'
import {
  INTERFACE_WAIT_MS,
  createPluginLoader,
  getPluginStyle,
  type PluginLoaderDeps,
  type LinkClient,
} from '@numen/plugin-host'
import type { VaultRow } from '@numen/shell'
import { formatErrorMessage } from '@numen/core-client'
import { runInBackground } from '@numen/ui'
import { agentService, noteService, pluginService, vaultsService } from '@/shared/clients'
import { createWindowAgent, type WindowAgentClient } from './agent'
import { createShellHost, type ShellServices } from './host'
import { useWindowFrame, type WindowFrame } from './useWindowFrame'
import { createWindowHost } from './windowHost'
import { createWindowVaults } from './windowVaults'
import { createWindowCommands } from './windowCommands'
import { createWindowKeys } from './windowKeys'
import { useExternalLinks } from './useExternalLinks'
import { useWindowTasks, WINDOW_SOURCE } from './useWindowTasks'
import { useAppearance, type AppearanceServices } from './useAppearance'

/** What the shell is given: the plugins it loads, how a module is fetched, and the vaults. */
export interface ShellDeps {
  readonly listPlugins?: () => Promise<readonly string[]>
  readonly listVaults?: () => Promise<readonly VaultRow[]>
  readonly fetch?: PluginLoaderDeps['fetch']
  readonly dynamicImport?: PluginLoaderDeps['dynamicImport']
  readonly agentClient?: WindowAgentClient
  readonly linkClient?: LinkClient
  /** The theme service and the plugin names the appearance commands are given. */
  readonly appearance?: AppearanceServices
}

const listPlugins = async (): Promise<readonly string[]> =>
  (await pluginService.getPlugins({})).plugins.map((one) => one.id)

const listVaults = async (): Promise<readonly VaultRow[]> =>
  (await vaultsService.listVaults({})).vaults.map((one) => ({
    id: one.id,
    name: one.name,
    path: one.path,
  }))

/** The vaults of the installation, read once as the window opens. */
function useVaults(list: () => Promise<readonly VaultRow[]>, onError: (text: string) => void) {
  const vaults = shallowRef<readonly VaultRow[]>([])
  const isListing = ref(true)

  onMounted(async () => {
    try {
      vaults.value = await list()
    } catch (error) {
      onError(formatErrorMessage(error))
    } finally {
      isListing.value = false
    }
  })

  return { vaults, isListing }
}

/** The commands, the vaults and the window the plugins of the window read. */
function fillServices(
  services: ShellServices,
  window: Pick<ReturnType<typeof createShellHost>, 'tab' | 'openTab' | 'notices'> & {
    readonly vaults: ReturnType<typeof useVaults>['vaults']
    readonly frame: WindowFrame
  },
): void {
  const { tab, notices, frame } = window
  Object.assign(services, {
    commands: createWindowCommands(),
    vaults: createWindowVaults({
      list: window.vaults,
      getTabVault: () => tab.value?.vault ?? null,
      openTab: window.openTab,
    }),
    window: createWindowHost(frame, (error) =>
      notices.show(WINDOW_SOURCE, formatErrorMessage(error), 'error'),
    ),
  })
}

export function useShell(deps: ShellDeps = {}) {
  const windowAgent = createWindowAgent(
    deps.agentClient ?? agentService,
    deps.linkClient ?? noteService,
  )
  const services: ShellServices = {}
  const frame = useWindowFrame()
  const { host, front, tab, openTab, notices, mountView, titleBarView } = createShellHost(
    windowAgent.agent,
    services,
    () => frame.platform.value,
  )
  const loader = createPluginLoader({
    host,
    ...(deps.fetch ? { fetch: deps.fetch } : {}),
    ...(deps.dynamicImport ? { dynamicImport: deps.dynamicImport } : {}),
  })
  const isLoading = ref(true)
  const listError = ref('')
  const { vaults, isListing } = useVaults(deps.listVaults ?? listVaults, (text) =>
    notices.show(WINDOW_SOURCE, text, 'error'),
  )
  fillServices(services, { vaults, tab, openTab, notices, frame })
  const onKeyDown = createWindowKeys({
    getTab: () => tab.value,
    getVaults: () => (front.value ? vaults.value : []),
    openVault: openTab,
  })
  useWindowTasks(notices)
  useExternalLinks(notices)
  const appearance = useAppearance(notices, deps.appearance)

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
      await loader.trackPlugins(await (deps.listPlugins ?? listPlugins)())
      await loader.start()
    } catch (error) {
      listError.value = formatErrorMessage(error)
    } finally {
      isLoading.value = false
      onStarted()
    }
  })
  onUnmounted(() => {
    clearTimeout(startTimer)
    window.removeEventListener('keydown', onKeyDown)
    loader.close().catch((error: unknown) => {
      listError.value = formatErrorMessage(error)
    })
    windowAgent.dispose()
  })

  /** Why the window draws no view: the plugins could not be listed, or each one failed to start. */
  const failures = computed<readonly string[]>(() => [
    ...(listError.value ? [listError.value] : []),
    ...loader.getFailedPlugins().map((one) => `${one.id}: ${one.error}`),
  ])

  const style = computed(() => (front.value ? getPluginStyle(front.value.pluginId) : undefined))

  return {
    front,
    tab,
    openTab,
    vaults: { list: vaults, isListing },
    plugins: { style, failures, isLoading, isStarted, titleBarView },
    notices,
    mountView,
    appearance,
  }
}
