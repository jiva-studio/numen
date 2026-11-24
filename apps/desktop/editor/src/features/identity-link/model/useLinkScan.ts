import { onMounted, ref } from 'vue'

/** Whether the scan form reads the camera, or, without a working camera, the field to paste the link into. */
export function useLinkScan(findCamera: () => Promise<boolean>) {
  const phase = ref<'checking' | 'camera' | 'paste'>('checking')

  async function detect(): Promise<void> {
    if (await findCamera()) {
      phase.value = 'camera'
      return
    }
    phase.value = 'paste'
  }

  function setUnavailable(): void {
    phase.value = 'paste'
  }

  onMounted(detect)

  return { phase, setUnavailable }
}
