/** Where the site points to. */
export const DOCS = 'https://docs.numen.md'
export const FORUM = 'https://forum.numen.md'
export const GITHUB = 'https://github.com/jiva-studio/numen/'
export const STUDIO = 'https://jiva.studio'
export const DOWNLOAD = '/#get'

/** The address of the latest release, from the configured base or the public default. */
export function getShelfBase(configured: string | undefined): string {
  const base = configured?.replace(/\/+$/, '')
  if (!base) return 'https://dl.numen.md/latest'
  return base.endsWith('/latest') ? base : `${base}/latest`
}

export const SHELF = getShelfBase(import.meta.env.PUBLIC_NUMEN_DOWNLOADS_BASE)

export const BUILDS = [
  {
    id: 'macos',
    name: 'macOS',
    takes: [{ name: 'Universal', note: 'Apple Silicon & Intel (.dmg)', file: 'numen-macos.dmg' }],
  },
  {
    id: 'windows',
    name: 'Windows',
    takes: [
      { name: 'Installer', note: 'x64 (.exe)', file: 'numen-windows-setup.exe' },
      { name: 'Portable', note: 'x64 (.zip)', file: 'numen-windows-amd64.zip' },
    ],
  },
  {
    id: 'linux',
    name: 'Linux',
    takes: [
      { name: 'Debian / Ubuntu', note: '.deb', file: 'numen-linux-amd64.deb' },
      { name: 'Fedora / RHEL', note: '.rpm', file: 'numen-linux-amd64.rpm' },
      { name: 'Flatpak', note: 'any distro', file: 'numen-linux-amd64.flatpak' },
      { name: 'Snap', note: 'Ubuntu', file: 'numen-linux-amd64.snap' },
      { name: 'Archive', note: '.tar.gz', file: 'numen-linux-amd64.tar.gz' },
    ],
  },
]

/** The path of the GitHub mark, in a 16 by 16 box. */
export const GITHUB_ICON =
  'M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z'
