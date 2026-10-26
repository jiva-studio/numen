/**
 * The shell put together: the plugins of the window's target, loaded over HTTP and activated, and
 * the view the window draws from them.
 */
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { createPluginLoader, getPluginStyle, type PluginLoaderDeps } from '@numen/plugin-host'
import { formatErrorMessage } from '@numen/wire'
import { pluginService } from '@/shared/clients'
import { createShellHost } from './host'

/** What the shell is given: the ids of the plugins it loads, and how a module is fetched. */
export interface ShellDeps {
  readonly listPlugins?: () => Promise<readonly string[]>
  readonly fetch?: PluginLoaderDeps['fetch']
  readonly dynamicImport?: PluginLoaderDeps['dynamicImport']
}

const listPlugins = async (): Promise<readonly string[]> =>
  (await pluginService.getPlugins({})).plugins.map((one) => one.id)

export function useShell(deps: ShellDeps = {}) {
  const { host, front } = createShellHost()
  const loader = createPluginLoader({
    host,
    ...(deps.fetch ? { fetch: deps.fetch } : {}),
    ...(deps.dynamicImport ? { dynamicImport: deps.dynamicImport } : {}),
  })
  const isLoading = ref(true)
  const listError = ref('')

  onMounted(async () => {
    try {
      await loader.trackPlugins(await (deps.listPlugins ?? listPlugins)())
      await loader.start()
    } catch (error) {
      listError.value = formatErrorMessage(error)
    } finally {
      isLoading.value = false
    }
  })
  onUnmounted(() => void loader.close())

  /** Why the window draws no view: the plugins could not be listed, or each one failed to start. */
  const failures = computed<readonly string[]>(() => [
    ...(listError.value ? [listError.value] : []),
    ...loader.getFailedPlugins().map((one) => `${one.id}: ${one.error}`),
  ])

  const style = computed(() => (front.value ? getPluginStyle(front.value.pluginId) : undefined))

  return { front, style, failures, isLoading }
}
