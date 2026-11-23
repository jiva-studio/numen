/** What the pages of linking show, read as values: fingerprints, codes and counts, apart from the page that shows them. */
import { createHash } from 'node:crypto'

const DEVICE_LABEL = 'numen/v1/fingerprint/device'
const CODE_DIGITS = 6
const FINGERPRINT_GROUPS = 8

const CODE_PATTERN = new RegExp(`\\b\\d{${CODE_DIGITS}}\\b`)
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

/** The six digits a page shows as the confirmation code. */
export const readCode = (text: string): string => {
  const found = CODE_PATTERN.exec(text)
  if (found === null) throw new Error(`"${text}" holds no confirmation code of ${CODE_DIGITS} digits`)
  return found[0]
}

/** The first fingerprint of eight groups of four hexadecimal digits in a text. */
export const readFingerprint = (text: string): string => {
  const found = FINGERPRINT_PATTERN.exec(text)
  if (found === null) throw new Error(`"${text}" holds no fingerprint`)
  return found[0]
}

/** A code of the same length as the right one, that is not it: the first digit moves by one. */
export const makeWrongCode = (code: string): string => {
  readCode(code)
  const first = (Number(code[0]) + 1) % 10
  return `${first}${code.slice(1)}`
}
