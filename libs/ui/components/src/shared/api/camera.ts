import { nav } from './dom'

/** Opens the camera of a device. The scanner asks the machine for one unless a caller gives its own. */
export type ScannerCamera = () => Promise<MediaStream>

/** How many frames a second the scanner reads. */
export const FRAMES_PER_SECOND = 8

/** Opens the camera that faces away from the person. */
export const openMachineCamera: ScannerCamera = () =>
  nav.mediaDevices.getUserMedia({ video: { facingMode: 'environment' } })

/** Whether this machine can open a camera at all. */
export const hasCamera = (): boolean => 'mediaDevices' in nav && typeof nav.mediaDevices.getUserMedia === 'function'

/** What a person reads when the camera does not open. */
export const describeCameraFailure = (cause: unknown): string => {
  const name = cause instanceof DOMException ? cause.name : ''
  if (name === 'NotAllowedError') return 'The camera is not allowed.'
  if (name === 'NotFoundError') return 'No camera was found.'
  return 'The camera did not open.'
}
