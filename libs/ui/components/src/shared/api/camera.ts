import { nav } from './dom'

/** Opens the camera of a device. The scanner asks the machine for one unless a caller gives its own. */
export type ScannerCamera = () => Promise<MediaStream>

/** How many frames a second the scanner reads. */
export const FRAMES_PER_SECOND = 8

/** Opens the camera that faces away from the person. */
export const openMachineCamera: ScannerCamera = async () =>
  nav.mediaDevices.getUserMedia({ video: { facingMode: 'environment' } })

/** Whether this machine has a camera to open. A machine that cannot list its devices is taken to have one. */
export const hasCamera = async (): Promise<boolean> => {
  if (!('mediaDevices' in nav) || typeof nav.mediaDevices.getUserMedia !== 'function') return false
  if (typeof nav.mediaDevices.enumerateDevices !== 'function') return true
  try {
    const devices = await nav.mediaDevices.enumerateDevices()
    return devices.some((device) => device.kind === 'videoinput')
  } catch {
    return true
  }
}

/** What a person reads when the camera does not open. */
export const describeCameraFailure = (cause: unknown): string => {
  const name = cause instanceof DOMException ? cause.name : ''
  if (name === 'NotAllowedError') return 'The camera is not allowed.'
  if (name === 'NotFoundError') return 'No camera found.'
  if (name === 'NotReadableError') return 'The camera is used by another program.'
  return 'The camera did not open.'
}
