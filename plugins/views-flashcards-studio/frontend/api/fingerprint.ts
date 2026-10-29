/**
 * The version of a file as the one string the window carries it: size, modification time and path.
 */
import type { FileStat } from './types'

/** The file a response came out of as one string, and nothing where the response names none. */
export const formatFingerprint = (at?: FileStat): string | undefined =>
  at && `${at.size} ${at.mtime} ${at.path}`

/** The two numbers first: a path holds spaces, and everything after them is it. */
export const parseFingerprint = (fingerprint: string): FileStat => {
  const [size = '0', mtime = '0', ...rest] = fingerprint.split(' ')
  return { path: rest.join(' '), size: Number(size), mtime }
}
