/**
 * The interface and text scales a story sets on the document, as the served page sets them.
 * Storybook test fixture only; it is not shipped.
 */

/** The scales, as [interface, text]: the designed size, each one larger, and both larger. */
export const SCALES: readonly (readonly [string, string])[] = [
  ['1', '1'],
  ['2', '1'],
  ['1', '2'],
  ['2', '2'],
]

export const setScales = (interfaceScale: string, textScale: string): void => {
  const style = document.documentElement.style
  style.setProperty('--numen-interface-scale', interfaceScale)
  style.setProperty('--numen-text-scale', textScale)
}

export const resetScales = (): void => {
  const style = document.documentElement.style
  style.removeProperty('--numen-interface-scale')
  style.removeProperty('--numen-text-scale')
}

/** Runs `measure` once at each scale and returns the readings in the order of `SCALES`. */
export const measureAtEveryScale = <T>(measure: () => T): T[] => {
  const found: T[] = []
  try {
    for (const [interfaceScale, textScale] of SCALES) {
      setScales(interfaceScale, textScale)
      found.push(measure())
    }
  } finally {
    resetScales()
  }
  return found
}
