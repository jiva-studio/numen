import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

/** One class list out of many, with the last word on a property winning. */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
