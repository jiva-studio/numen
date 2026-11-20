import { computed, shallowRef, watch } from 'vue'
import type { ProcessHost } from '@numen/plugin-loader'
import { WORDS } from '../words'

/** Follows the model sessions of the process selected in settings. */
export function useModelProviders(
  getPluginId: () => string,
  getProcesses: () => ProcessHost,
  getOptions: () => readonly { readonly value: string; readonly title: string }[],
) {
  const status = shallowRef('')
  const sessions = shallowRef<readonly { readonly stage: string; readonly provider: string }[]>([])
  watch(
    [getPluginId, getProcesses],
    ([pluginId, processes], _, onCleanup) => {
      status.value = ''
      sessions.value = []
      const subscription = processes.watch((running) => {
        const selected = running.find((process) => process.pluginId === pluginId)
        sessions.value = selected?.modelProviders ?? []
        status.value = selected ? WORDS.noModelLoaded : WORDS.pluginStopped
      })
      onCleanup(() => subscription.dispose())
    },
    { immediate: true },
  )
  const providers = computed(() =>
    sessions.value.map(({ stage, provider }) => ({
      key: JSON.stringify([stage, provider]),
      stage,
      provider: getOptions().find((option) => option.value === provider)?.title ?? provider,
    })),
  )
  return { status, providers }
}
