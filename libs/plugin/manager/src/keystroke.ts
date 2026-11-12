/** A shortcut as a manifest writes it, such as `mod+shift+a`, read into what a key cap draws. */
import type { Keystroke } from '@numen/ui'

type KeyIcon = Keystroke['icons'][number]

const MODIFIERS: Readonly<Record<string, (isApple: boolean) => KeyIcon>> = {
  mod: (isApple) => (isApple ? 'command' : 'control'),
  ctrl: () => 'control',
  shift: () => 'shift',
  alt: () => 'option',
}

/**
 * The keystroke of a shortcut. `mod` is Command on Apple systems and Control elsewhere; a modifier
 * of another name is left out.
 */
export const parseKeystroke = (keys: string, isApple: boolean): Keystroke => {
  const parts = keys.split('+')
  const icons = parts.slice(0, -1).flatMap((one) => {
    const icon = MODIFIERS[one.toLowerCase()]
    return icon ? [icon(isApple)] : []
  })
  return { icons, letter: (parts.at(-1) ?? '').toUpperCase() }
}
