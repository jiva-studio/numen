const DEFAULT_FOLDER = 'Templates'

/** The folder the templates setting names, as a vault path; Templates when it names none. */
export function normalizeFolder(raw: string): string {
  const segments = raw
    .trim()
    .replaceAll('\\', '/')
    .split('/')
    .filter((segment) => segment !== '' && segment !== '.')
  if (segments.length === 0 || segments.includes('..')) return DEFAULT_FOLDER
  return segments.join('/')
}
