import { clsx, type ClassValue } from 'clsx'
import { extendTailwindMerge } from 'tailwind-merge'

/**
 * `text-small` is a font size of the theme. Unknown to the merge, it is read as a text colour and
 * removes the colour a variant sets before it.
 */
const merge = extendTailwindMerge({
  extend: { classGroups: { 'font-size': [{ text: ['small'] }] } },
})

/** One class list out of many, with the last word on a property winning. */
export function cn(...inputs: ClassValue[]) {
  return merge(clsx(inputs))
}
