/**
 * The keys of a keybinding: a chord such as `mod+shift+p` read once into its modifiers and the
 * physical key it names. A key is matched by KeyboardEvent.code, so `p` is the key at the place of P
 * in any layout, the Russian one included.
 */

/**
 * A chord read once: the modifiers it needs held, and the `code` of its key. `mod` stands for Meta on
 * macOS and Ctrl elsewhere; the platform is known when an event is matched, not when the chord is read.
 */
export interface ParsedKey {
  readonly mod: boolean
  readonly ctrl: boolean
  readonly meta: boolean
  readonly alt: boolean
  readonly shift: boolean
  readonly code: string
}

const CODES: Readonly<Record<string, string>> = {
  enter: 'Enter',
  escape: 'Escape',
  tab: 'Tab',
  space: 'Space',
  backspace: 'Backspace',
  delete: 'Delete',
  home: 'Home',
  end: 'End',
  pageup: 'PageUp',
  pagedown: 'PageDown',
  up: 'ArrowUp',
  down: 'ArrowDown',
  left: 'ArrowLeft',
  right: 'ArrowRight',
  comma: 'Comma',
  period: 'Period',
  slash: 'Slash',
  backslash: 'Backslash',
  semicolon: 'Semicolon',
  quote: 'Quote',
  minus: 'Minus',
  equal: 'Equal',
  backquote: 'Backquote',
  bracketleft: 'BracketLeft',
  bracketright: 'BracketRight',
}

const MODIFIERS: ReadonlySet<string> = new Set(['mod', 'ctrl', 'alt', 'shift', 'meta'])

function getCode(name: string): string | undefined {
  if (/^[a-z]$/.test(name)) return `Key${name.toUpperCase()}`
  if (/^[0-9]$/.test(name)) return `Digit${name}`
  if (/^f([1-9]|1[0-2])$/.test(name)) return name.toUpperCase()
  return Object.hasOwn(CODES, name) ? CODES[name] : undefined
}

/**
 * The chord read. Undefined where a part is not a modifier (mod, ctrl, alt, shift, meta) or the last
 * part is not a known key name.
 */
export function parseKey(text: string): ParsedKey | undefined {
  const parts = text
    .toLowerCase()
    .split('+')
    .map((part) => part.trim())
  const name = parts.pop() ?? ''
  const code = getCode(name)
  if (code === undefined || parts.some((part) => !MODIFIERS.has(part))) return undefined
  const has = (modifier: string) => parts.includes(modifier)
  return {
    mod: has('mod'),
    ctrl: has('ctrl'),
    meta: has('meta'),
    alt: has('alt'),
    shift: has('shift'),
    code,
  }
}

function getEventCode(event: KeyboardEvent): string {
  if (event.code) return event.code
  return getCode(event.key.toLowerCase()) ?? ''
}

/**
 * Whether the event is the chord: its physical key, with exactly the modifiers the chord needs held,
 * where `mod` is Meta if `isMac` and Ctrl if not.
 */
export function isKeyEvent(parsed: ParsedKey, event: KeyboardEvent, isMac: boolean): boolean {
  return (
    getEventCode(event) === parsed.code &&
    event.ctrlKey === (parsed.ctrl || (parsed.mod && !isMac)) &&
    event.metaKey === (parsed.meta || (parsed.mod && isMac)) &&
    event.altKey === parsed.alt &&
    event.shiftKey === parsed.shift
  )
}
