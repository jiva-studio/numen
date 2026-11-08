import type { AppInfo } from '@numen/plugin-sdk'

/** The label of each modifier of a shortcut, as a keyboard of the platform prints it. */
const MODIFIERS: Readonly<Record<string, { readonly mac: string; readonly other: string }>> = {
  mod: { mac: '⌘', other: 'Ctrl' },
  cmd: { mac: '⌘', other: 'Ctrl' },
  ctrl: { mac: 'Ctrl', other: 'Ctrl' },
  control: { mac: 'Ctrl', other: 'Ctrl' },
  shift: { mac: 'Shift', other: 'Shift' },
  alt: { mac: 'Alt', other: 'Alt' },
  option: { mac: 'Alt', other: 'Alt' },
}

const formatKey = (key: string): string => key.charAt(0).toUpperCase() + key.slice(1)

/** A shortcut such as `mod+shift+e` as the person reads it on the platform; empty for none. */
export function formatShortcut(shortcut: string, platform: AppInfo['platform']): string {
  const keys = shortcut
    .toLowerCase()
    .split('+')
    .map((key) => key.trim())
    .filter((key) => key !== '')
  return keys
    .map((key) => {
      const modifier = MODIFIERS[key]
      if (!modifier) return formatKey(key)
      return platform === 'darwin' ? modifier.mac : modifier.other
    })
    .join('+')
}
