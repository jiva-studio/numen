import { shallowRef } from 'vue'
import type { DialogHost, DialogOptions } from '@numen/plugin-loader'

export interface OpenDialog {
  readonly id: number
  readonly options: DialogOptions
}

/** The dialogs the plugins have open, in the order they opened, and the host that opens and closes them. */
export function createDialogStack() {
  const entries = shallowRef<readonly OpenDialog[]>([])
  let last = 0
  const host: DialogHost = {
    open: (options) => {
      const id = ++last
      entries.value = [...entries.value, { id, options }]
      return {
        close: () => {
          entries.value = entries.value.filter((one) => one.id !== id)
        },
      }
    },
  }
  return { entries, host }
}

export type DialogStack = ReturnType<typeof createDialogStack>
