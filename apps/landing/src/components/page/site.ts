/** Where the site points to. */
export const DOCS = 'https://docs.numen.md'
export const FORUM = 'https://forum.numen.md'
export const GITHUB = 'https://github.com/jiva-studio/numen/'
export const STUDIO = 'https://jiva.studio'
export const DOWNLOAD = '#get'

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
