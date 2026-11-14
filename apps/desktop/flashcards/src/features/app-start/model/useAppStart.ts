import { computed, onMounted, onUnmounted, ref } from 'vue'
import { waitWithinInterfaceLimit } from '@numen/plugin-loader'
import { runInBackground } from '@numen/ui'
import type { PluginLister } from '@/entities/plugin'
import { formatError, type Caught } from '@/shared/api/errors'

/** What the start needs of the plugin loader. */
export interface PluginStarter {
  readonly trackPlugins: (ids: readonly string[]) => Promise<void>
  readonly start: () => Promise<void>
  readonly close: () => Promise<void>
  readonly status: {
    readonly waitForInterface: () => Promise<void>
    readonly startPlugin: (id: string) => Promise<void>
    readonly getFailedPlugins: () => readonly { readonly id: string; readonly error: string }[]
  }
}

/**
 * Starts the plugins `listPlugins` lists when the window mounts and closes them when it unmounts.
 * The window is started once the plugins of its slots have started or INTERFACE_WAIT_MS has passed
 * since it mounted.
 */
export function useAppStart(listPlugins: PluginLister, loader: PluginStarter) {
  const isLoading = ref(true)
  const isStarted = ref(false)
  const listError = ref('')
  const unmounting = new AbortController()

  const onStarted = (): void => {
    isStarted.value = true
  }

  onMounted(async () => {
    runInBackground(
      waitWithinInterfaceLimit(loader.status.waitForInterface(), unmounting.signal).then(onStarted),
      () => undefined,
    )
    try {
      const ids = (await listPlugins()).map((one) => one.id)
      await loader.trackPlugins(ids)
      await loader.start()
      // The window draws the tab kinds and the slot entries of the plugins it started, so it starts
      // every plugin it lists rather than each on use.
      await Promise.all(ids.map((id) => loader.status.startPlugin(id)))
    } catch (error) {
      listError.value = formatError(error)
    } finally {
      isLoading.value = false
      onStarted()
    }
  })
  onUnmounted(() => {
    unmounting.abort()
    loader.close().catch((error: Caught) => {
      listError.value = formatError(error)
    })
  })

  /** Why the window draws no view: the plugins could not be listed, or each one failed to start. */
  const failures = computed<readonly string[]>(() => [
    ...(listError.value ? [listError.value] : []),
    ...loader.status.getFailedPlugins().map((one) => `${one.id}: ${one.error}`),
  ])

  return { isLoading, isStarted, failures }
}
