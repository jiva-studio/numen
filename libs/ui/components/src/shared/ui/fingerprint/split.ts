/** How many trailing groups stay visible when the value is cut in the middle. */
const TAIL_GROUPS = 2

export interface FingerprintParts {
  head: string
  tail: string
}

/** Splits a value of space-separated groups into the part that may be cut and the last groups that never are. */
export function splitFingerprint(value: string): FingerprintParts {
  const groups = value.trim().split(/\s+/).filter(Boolean)
  if (groups.length <= TAIL_GROUPS) return { head: '', tail: groups.join(' ') }
  return {
    head: groups.slice(0, -TAIL_GROUPS).join(' '),
    tail: ` ${groups.slice(-TAIL_GROUPS).join(' ')}`,
  }
}
