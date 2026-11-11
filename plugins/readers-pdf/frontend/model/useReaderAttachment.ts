import { onMounted, onUnmounted } from 'vue'
import type { ReaderHandle, ReaderTabState } from '@numen/plugin-sdk'

/** Focuses the reader and hands the window its handle from mount, and takes the handle back at unmount, where `onDetach` also runs. */
export function useReaderAttachment(
  getState: () => Pick<ReaderTabState, 'attachReader'>,
  reader: ReaderHandle,
  onDetach: () => void,
): void {
  onMounted(() => {
    reader.focus()
    getState().attachReader?.(reader)
  })

  onUnmounted(() => {
    getState().attachReader?.(null)
    onDetach()
  })
}
