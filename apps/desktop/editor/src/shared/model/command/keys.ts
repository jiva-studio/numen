/**
 * The keys drawn beside a command: those of the application come from its keybindings, those of a
 * plugin from the key the host holds for its command. Control and Command are equivalent.
 */
import { APP_KEYBINDINGS } from '@numen/plugin-loader'
import { keyChord, type Keystroke } from '@numen/ui'

/** The keys that hold the window's modifier in a key a plugin declares, such as `mod+shift+x`. */
const MODIFIERS: ReadonlySet<string> = new Set(['mod', 'ctrl', 'control', 'cmd'])

/**
 * The keys of a declared key as drawn for the current platform, or undefined unless the key is the
 * window's modifier plus a letter, optionally with Shift.
 */
export const findShortcutKeys = (shortcut: string, agent: string): Keystroke | undefined => {
  const parts = shortcut
    .toLowerCase()
    .split('+')
    .map((part) => part.trim())
  const letter = parts.at(-1) ?? ''
  const modifiers = parts.slice(0, -1)
  if (!/^[a-z]$/.test(letter) || !modifiers.some((one) => MODIFIERS.has(one))) return undefined
  if (modifiers.some((one) => !MODIFIERS.has(one) && one !== 'shift')) return undefined
  return keyChord(letter, agent, modifiers.includes('shift'))
}

type Values = Readonly<Record<string, string>>

const hasSameValues = (a: Values, b: Values): boolean => {
  const names = Object.keys(a)
  return names.length === Object.keys(b).length && names.every((name) => a[name] === b[name])
}

const findChordKeys = (
  command: string,
  agent: string,
  values: Values = {},
): Keystroke | undefined => {
  const key = APP_KEYBINDINGS.find(
    (one) => one.command === command && hasSameValues(one.values, values),
  )?.key
  return key === undefined ? undefined : findShortcutKeys(key, agent)
}

/** The keys of the command with these values as a palette row property. */
export const createRowKeys = (
  command: string,
  agent: string,
  values: Values = {},
): { keys?: Keystroke } => {
  const keys = findChordKeys(command, agent, values)
  return keys ? { keys } : {}
}
