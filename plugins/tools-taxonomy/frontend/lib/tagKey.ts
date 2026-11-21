/**
 * What two spellings of one tag share, as the backend compares them: without the spaces around it and
 * a leading "#", in Unicode NFC, each segment between "/" trimmed and the empty ones dropped, and the
 * case folded.
 */
export function getTagKey(tag: string): string {
  return tag
    .trim()
    .replace(/^#/, '')
    .normalize('NFC')
    .split('/')
    .map((segment) => segment.trim())
    .filter((segment) => segment !== '')
    .join('/')
    .toLowerCase()
}
