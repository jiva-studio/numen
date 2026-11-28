/** What the pages of linking show, read as values: fingerprints and counts, apart from the page that shows them. */
import { createHash } from 'node:crypto'

const DEVICE_LABEL = 'numen/v1/fingerprint/device'
const FINGERPRINT_GROUPS = 8

const FINGERPRINT_PATTERN = /\b(?:[0-9A-F]{4} ){7}[0-9A-F]{4}\b/

/** The fingerprint of a device as a page shows it, from the hexadecimal signing key a sync server lists in a session. */
export const getDeviceFingerprint = (signKeyHex: string): string => {
  const key = Buffer.from(signKeyHex, 'hex')
  const sum = createHash('sha256')
    .update(Buffer.concat([Buffer.from(DEVICE_LABEL, 'utf8'), Buffer.from([0]), key]))
    .digest('hex')
    .slice(0, 32)
    .toUpperCase()
  const groups = sum.match(/.{4}/g)
  if (groups?.length !== FINGERPRINT_GROUPS) throw new Error(`the signing key "${signKeyHex}" has no fingerprint`)
  return groups.join(' ')
}

/** The first fingerprint of eight groups of four hexadecimal digits in a text. */
export const readFingerprint = (text: string): string => {
  const found = FINGERPRINT_PATTERN.exec(text)
  if (found === null) throw new Error(`"${text}" holds no fingerprint`)
  return found[0]
}

