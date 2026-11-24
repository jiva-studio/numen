import { onMounted, ref } from 'vue'

/** How long the check of the camera may take before the dialog shows the field to paste the link into. */
const CAMERA_CHECK_MS = 2000

function findCameraInTime(findCamera: () => Promise<boolean>): Promise<boolean> {
  return new Promise((resolve) => {
    const timer = setTimeout(() => resolve(false), CAMERA_CHECK_MS)
    const settle = (found: boolean): void => {
      clearTimeout(timer)
      resolve(found)
    }
    findCamera().then(settle, () => settle(false))
  })
}

/** Whether the scan form reads the camera, or, without a working camera, the field to paste the link into. */
export function useLinkScan(findCamera: () => Promise<boolean>) {
  const phase = ref<'checking' | 'camera' | 'paste'>('checking')

  async function detect(): Promise<void> {
    if (await findCameraInTime(findCamera)) {
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
