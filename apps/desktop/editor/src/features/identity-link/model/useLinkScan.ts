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

/**
 * Whether the scan form reads the camera, or, without a working camera, the field to paste the link into.
 * A restart opens the camera again: the scanner stops after the first code it reads.
 */
export function useLinkScan(findCamera: () => Promise<boolean>) {
  const phase = ref<'checking' | 'camera' | 'paste'>('checking')
  const attempt = ref(0)

  async function detect(): Promise<void> {
    if (await findCameraInTime(findCamera)) {
      phase.value = 'camera'
      return
    }
    phase.value = 'paste'
  }

  function restart(): void {
    attempt.value += 1
  }

  function setUnavailable(): void {
    phase.value = 'paste'
  }

  onMounted(detect)

  return { phase, attempt, restart, setUnavailable }
}
