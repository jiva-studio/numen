/**
 * The version of a file as the one string the window carries it: `<size>:<version>:<hash> <mtime> <path>`.
 */
import type { FileStat } from '../lib/wire/notes'

/** The file a response came out of as one string, and nothing where the response names none. */
export const formatFingerprint = (at?: FileStat): string | undefined =>
  at && `${at.size}:${at.version}:${at.hash} ${at.mtime} ${at.path}`

/** The first two parts are `<size>:<version>:<hash> <mtime>`: a path holds spaces, and everything after them is it. */
export const parseFingerprint = (fingerprint: string): FileStat => {
  const [head = '', mtime, ...rest] = fingerprint.split(' ')
  const [size, version, hash, ...extra] = head.split(':')
  if (
    size === undefined ||
    version === undefined ||
    hash === undefined ||
    extra.length > 0 ||
    mtime === undefined ||
    rest.join(' ') === ''
  )
    throw new Error(`not the fingerprint of a file: ${fingerprint}`)
  return { path: rest.join(' '), size: Number(size), mtime, hash, version }
}
