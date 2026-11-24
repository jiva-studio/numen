const PREFIX = 'numen://share/'

/** The link a person sends to someone who wants to share a folder with them. It holds the identity code only. */
export const getSharingLink = (code: string): string => `${PREFIX}${code}`

/** The identity code in a sharing link; a text that is a bare code is returned as it is. */
export function getCodeFromLink(text: string): string {
  const trimmed = text.trim()
  return trimmed.startsWith(PREFIX) ? trimmed.slice(PREFIX.length) : trimmed
}
