const SHORT_GROUPS = 3

/** The first groups of a fingerprint, enough to tell two people apart, with an ellipsis when cut. */
export function getShortFingerprint(fingerprint: string): string {
  const groups = fingerprint.trim().split(/\s+/)
  if (groups.length <= SHORT_GROUPS) return groups.join(' ')
  return `${groups.slice(0, SHORT_GROUPS).join(' ')}…`
}
