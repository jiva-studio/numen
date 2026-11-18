import { ref, type Ref } from 'vue'
import type { Disposable, PluginLicenses } from '@numen/plugin-sdk'

export const SUPPORTER_LICENSE_ID = 'supporter'

export interface SupporterState {
  readonly isActive: Readonly<Ref<boolean>>
  start(): void
  stop(): void
}

export function createSupporterState(licenses: PluginLicenses): SupporterState {
  const isActive = ref(licenses.has(SUPPORTER_LICENSE_ID))
  let watching: Disposable | undefined

  return {
    isActive,
    start: () => {
      isActive.value = licenses.has(SUPPORTER_LICENSE_ID)
      watching = licenses.onChange(SUPPORTER_LICENSE_ID, (next) => {
        isActive.value = next
      })
    },
    stop: () => {
      watching?.dispose()
      watching = undefined
    },
  }
}
