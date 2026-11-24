import { onMounted, ref } from 'vue'
import { WORDS as words } from '../words'

/** Whether the scan form reads the camera, and the notice that sends the person to the text field instead. */
export function useLinkScan(findCamera: () => Promise<boolean>) {
  const phase = ref<'checking' | 'camera' | 'paste'>('checking')
  const notice = ref('')

  async function detect(): Promise<void> {
    if (await findCamera()) {
      phase.value = 'camera'
      return
    }
    phase.value = 'paste'
    notice.value = words.linkNoCamera
  }

  function setUnavailable(reason: string): void {
    phase.value = 'paste'
    notice.value = words.linkCameraFailed(reason)
  }

  onMounted(detect)

  return { phase, notice, setUnavailable }
}
