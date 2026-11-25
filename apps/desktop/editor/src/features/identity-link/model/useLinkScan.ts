import { onBeforeUnmount, onMounted, ref } from 'vue'

/** How long the check of the camera may take before the dialog shows the field to paste the link into. */
const CAMERA_CHECK_MS = 2000

interface CameraCheck {
  found: Promise<boolean>
  cancel: () => void
}

function findCameraInTime(findCamera: () => Promise<boolean>): CameraCheck {
  let cancel = (): void => undefined
  const found = new Promise<boolean>((resolve) => {
    const timer = setTimeout(() => resolve(false), CAMERA_CHECK_MS)
    const settle = (isFound: boolean): void => {
      clearTimeout(timer)
      resolve(isFound)
    }
    cancel = () => settle(false)
    findCamera().then(settle, () => settle(false))
  })
  return { found, cancel: () => cancel() }
}

/**
 * Whether the scan form reads the camera, or, without a working camera, the field to paste the link into.
 * A restart opens the camera again: the scanner stops after the first code it reads.
 */
export function useLinkScan(findCamera: () => Promise<boolean>) {
  const phase = ref<'checking' | 'camera' | 'paste'>('checking')
  const attempt = ref(0)
  let isDisposed = false
  let cancelCheck = (): void => undefined

  async function detect(): Promise<void> {
    const check = findCameraInTime(findCamera)
    cancelCheck = check.cancel
    const isFound = await check.found
    if (isDisposed) return
    phase.value = isFound ? 'camera' : 'paste'
  }

  function restart(): void {
    attempt.value += 1
  }

  function setUnavailable(): void {
    phase.value = 'paste'
  }

  onMounted(detect)
  onBeforeUnmount(() => {
    isDisposed = true
    cancelCheck()
  })

  return { phase, attempt, restart, setUnavailable }
}
