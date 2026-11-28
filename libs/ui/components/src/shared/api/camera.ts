import { nav } from './dom'

/** Opens the camera of a device. The scanner asks the machine for one unless a caller gives its own. */
export type ScannerCamera = () => Promise<MediaStream>

/** How many frames a second the scanner reads. */
export const FRAMES_PER_SECOND = 8

/** Causes that mean the asked kind of camera is missing while another camera may still open. */
const MISSING_CAMERA_CAUSES = new Set(['OverconstrainedError', 'NotFoundError'])

/** Releases the camera: every track of the stream stops. */
export const stopStream = (stream: MediaStream): void => {
  stream.getTracks().forEach((track) => track.stop())
}

/** Opens the camera that faces away from the person, or any camera when the machine has no such one. */
export const openMachineCamera: ScannerCamera = async () => {
  try {
    return await nav.mediaDevices.getUserMedia({ video: { facingMode: { ideal: 'environment' } } })
  } catch (cause) {
    if (!(cause instanceof DOMException) || !MISSING_CAMERA_CAUSES.has(cause.name)) throw cause
    return nav.mediaDevices.getUserMedia({ video: true })
  }
}

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
  if (name === 'NotFoundError' || name === 'OverconstrainedError') return 'No camera found.'
  if (name === 'NotReadableError') return 'The camera is used by another program.'
  return 'The camera did not open.'
}
