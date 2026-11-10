/** What the goals of a vault come to today, one tile to a preset. */
import { percent } from '@numen/ui'

import { getSpentShare } from './progress'
import { getGoalWords, getLeftWords } from './words'
import type { Preset } from './types'

/** One preset's tile: its goal, and how far through the day it is. */
export interface Tile {
  readonly one: Preset
  readonly goal: string
  readonly wrong: string
  readonly says: string
  readonly left: string
  readonly isOver: boolean
  readonly canOpen: boolean
}

/**
 * A tile for each preset of the vault that schedules something, on the day it is
 * read on.
 */
export const getTiles = (presets: readonly Preset[], today: string): Tile[] =>
  presets.filter(isScheduling).map((one) => {
    const isDone = getSpentShare(one)
    const isOver = isDone > 1
    return {
      one,
      goal: one.paused || (one.settings ? getGoalWords(one.settings, today) : ''),
      wrong: one.wrong,
      says: isOver ? 'over budget' : percent(isDone),
      left: one.paused ? '' : getLeftWords(one),
      isOver,
      canOpen: !one.paused && one.cards > 0,
    }
  })

const isScheduling = (one: Preset): boolean => one.named > 0 && one.reviewCards > 0
