import type { TaskBatch } from '../lib/summary'

export const APPEAR_DELAY_MS = 300
export const HOLD_MS = 500

/** What the chip draws: the text and the ring fraction together. */
export interface ChipState extends TaskBatch {
  readonly isActive: boolean
}

export const IDLE_CHIP: ChipState = { isActive: false, done: 0, total: 0, fraction: 0 }

/** A chip display driven by the live state; stop() clears its timers. */
export interface ChipDisplay {
  update(live: ChipState): void
  stop(): void
}

const FIELDS = ['isActive', 'done', 'total', 'fraction'] as const

const isSame = (a: ChipState, b: ChipState): boolean =>
  FIELDS.every((field) => a[field] === b[field])

/**
 * Calls onChange with the state to draw. From idle, a busy period is drawn after APPEAR_DELAY_MS.
 * After a drawn change, the next one waits until HOLD_MS have passed and the latest value wins.
 */
export function createChipDisplay(
  onChange: (state: ChipState) => void,
  now: () => number,
): ChipDisplay {
  let live = IDLE_CHIP
  let visible = IDLE_CHIP
  let changedAt = -Infinity
  let appearTimer: ReturnType<typeof setTimeout> | undefined
  let holdTimer: ReturnType<typeof setTimeout> | undefined

  function clearAppear() {
    clearTimeout(appearTimer)
    appearTimer = undefined
  }

  function clearHold() {
    clearTimeout(holdTimer)
    holdTimer = undefined
  }

  function commit() {
    clearHold()
    const target = live.isActive ? live : IDLE_CHIP
    if (isSame(target, visible)) return
    const wait = changedAt + HOLD_MS - now()
    if (wait > 0) {
      holdTimer = setTimeout(commit, wait)
      return
    }
    visible = target
    changedAt = now()
    onChange(target)
  }

  function onAppear() {
    appearTimer = undefined
    commit()
  }

  return {
    update(next) {
      live = next
      if (!visible.isActive && live.isActive) {
        if (appearTimer === undefined && holdTimer === undefined) {
          appearTimer = setTimeout(onAppear, APPEAR_DELAY_MS)
        }
        return
      }
      clearAppear()
      commit()
    },
    stop() {
      clearAppear()
      clearHold()
    },
  }
}
