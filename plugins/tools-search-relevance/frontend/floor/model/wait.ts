import type { Wait } from './types'

/** The browser timer as a Wait. */
export const sleep: Wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms))
