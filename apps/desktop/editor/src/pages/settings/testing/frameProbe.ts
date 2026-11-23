import { measureRect } from '@numen/ui'

interface Frame {
  readonly n: number
  readonly top: number
  readonly classes: ReadonlySet<string>
  readonly fades: ReadonlySet<string>
  readonly isSameNode: boolean
}

const JUMP_OF_TOP = 11
const FLIP_WINDOW = 6
const LIFT_FADES = ['background-color', 'box-shadow', 'opacity']
const LIFTED = 'plugin-slots__row--lifted'
const STATE_CLASSES = [LIFTED, 'plugin-slots__row--settling']

const getLabelRows = (rows: readonly HTMLElement[]): (readonly [string, HTMLElement])[] => {
  let separators = 0
  return rows.map((row) => [
    row.querySelector('.plugin-slots__label')?.textContent ?? `|${separators++}`,
    row,
  ])
}

const getFades = (row: HTMLElement): Set<string> =>
  new Set(
    row
      .getAnimations()
      .flatMap((one) => (one instanceof CSSTransition ? [one.transitionProperty] : [])),
  )

/** Samples every row on every animation frame: where it is, which classes it has, which properties fade, and whether it is still the same element. */
export function startProbe(root: HTMLElement): { stop: () => Map<string, Frame[]> } {
  const listRows = (): HTMLElement[] =>
    Array.from(root.querySelectorAll<HTMLElement>('.plugin-slots__row'))
  const first = new Map(getLabelRows(listRows()))
  const frames = new Map<string, Frame[]>()
  let n = 0
  let isRunning = true
  const sample = (): void => {
    if (!isRunning) return
    getLabelRows(listRows()).forEach(([label, row]) => {
      frames.set(label, [
        ...(frames.get(label) ?? []),
        {
          n,
          top: measureRect(row).top,
          classes: new Set(row.classList),
          fades: getFades(row),
          isSameNode: first.get(label) === row,
        },
      ])
    })
    n += 1
    requestAnimationFrame(sample)
  }
  requestAnimationFrame(sample)
  return {
    stop: () => {
      isRunning = false
      return frames
    },
  }
}

const isFlipped = (list: readonly Frame[], at: number, name: string): boolean => {
  const has = (index: number): boolean => list[index]?.classes.has(name) === true
  for (let back = 1; back <= FLIP_WINDOW; back++) {
    if (has(at) && !has(at - back) && has(at - back - 1)) return true
  }
  return false
}

const isFading = (now: Frame): boolean =>
  now.classes.has(LIFTED) && LIFT_FADES.some((property) => now.fades.has(property))

/** Every frame where a row jumps, fades a property while lifted, is replaced, or has a state class flipped on and straight off. */
export function findDiscontinuities(frames: Map<string, Frame[]>): string[] {
  const found: string[] = []
  frames.forEach((list, label) => {
    list.forEach((now, at) => {
      const prev = list[at - 1]
      if (!prev) return
      const report = (what: string): number => found.push(`${label} f${now.n}: ${what}`)
      if (Math.abs(now.top - prev.top) > JUMP_OF_TOP) report(`top ${prev.top} -> ${now.top}`)
      if (isFading(now)) report('lifted row fades a property')
      if (now.isSameNode !== prev.isSameNode) report('element replaced')
      if (STATE_CLASSES.some((name) => isFlipped(list, at, name))) report('state class flips back')
    })
  })
  return found
}
