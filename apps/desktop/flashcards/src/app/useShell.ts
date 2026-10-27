/**
 * The shell put together: the plugins of the window's target, loaded over HTTP and activated; the
 * vaults a person chooses from; and the tab the window opens on the vault chosen.
 */
import { computed, onMounted, onUnmounted, ref, shallowRef } from 'vue'
import { createPluginLoader, getPluginStyle, type PluginLoaderDeps } from '@numen/plugin-host'
import type { VaultRow } from '@numen/shell'
import { formatErrorMessage } from '@numen/wire'
import { agentService, pluginService, vaultsService } from '@/shared/clients'
import { createWindowAgent, type WindowAgentClient } from './agent'
import { createShellHost } from './host'
import { createWindowKeys } from './windowKeys'
import { useWindowTasks, WINDOW_SOURCE } from './useWindowTasks'

/** What the shell is given: the plugins it loads, how a module is fetched, and the vaults. */
export interface ShellDeps {
  readonly listPlugins?: () => Promise<readonly string[]>
  readonly listVaults?: () => Promise<readonly VaultRow[]>
  readonly fetch?: PluginLoaderDeps['fetch']
  readonly dynamicImport?: PluginLoaderDeps['dynamicImport']
  readonly agentClient?: WindowAgentClient
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

export function useShell(deps: ShellDeps = {}) {
  const windowAgent = createWindowAgent(deps.agentClient ?? agentService)
  const { host, front, tab, openTab, notices, mountView } = createShellHost(windowAgent.agent)
  const loader = createPluginLoader({
    host,
    ...(deps.fetch ? { fetch: deps.fetch } : {}),
    ...(deps.dynamicImport ? { dynamicImport: deps.dynamicImport } : {}),
  })
  const isLoading = ref(true)
  const listError = ref('')
  const { vaults, isListing } = useVaults(deps.listVaults ?? listVaults, (text) =>
    notices.show(WINDOW_SOURCE, text, 'alarm'),
  )
  const onKeyDown = createWindowKeys({
    getTab: () => tab.value,
    getVaults: () => (front.value ? vaults.value : []),
    openVault: openTab,
  })
  useWindowTasks(notices)

  onMounted(async () => {
    window.addEventListener('keydown', onKeyDown)
    try {
      await loader.trackPlugins(await (deps.listPlugins ?? listPlugins)())
      await loader.start()
    } catch (error) {
      listError.value = formatErrorMessage(error)
    } finally {
      isLoading.value = false
    }
  })
  onUnmounted(() => {
    window.removeEventListener('keydown', onKeyDown)
    void loader.close()
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
    plugins: { style, failures, isLoading },
    notices,
    mountView,
  }
}
