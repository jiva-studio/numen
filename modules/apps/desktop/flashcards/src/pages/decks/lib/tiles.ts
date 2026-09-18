/** What the goals of a vault come to today, one tile to a preset. */
import { percent } from '@numen/ui'

import { getSpentShare } from './progress'
import { getGoalWords, getLeftWords } from '../words'
import type { Preset } from '../types'

/** One preset's tile: its goal, and how far through the day it stands. */
export interface Tile {
  readonly one: Preset
  /**
   * What stands under the name: the goal, or why it schedules nothing. A preset
   * scheduling nothing is asked for its reason, not for what it was aiming at.
   */
  readonly goal: string
  /** What is wrong with the preset, and empty where nothing is. */
  readonly wrong: string
  /** What stands at the right of the tile: the figure, or words in its place. */
  readonly says: string
  /**
   * What starting a session on it would ask, or why it would ask nothing. A preset
   * scheduling nothing has said why under its name, and stands here empty.
   */
  readonly left: string
  /** Whether the day is past its budget, which is the one thing to catch the eye. */
  readonly isOver: boolean
  /** Whether it has a session to offer, which is what makes the tile pressable. */
  readonly canStart: boolean
}

/**
 * A tile for each preset of the vault that schedules something, on the day it is
 * read on.
 */
export const getTiles = (presets: readonly Preset[], today: string): Tile[] =>
  presets.filter(isScheduling).map((one) => {
    const done = getSpentShare(one)
    const isOver = done > 1
    return {
      one,
      goal: one.paused || (one.settings ? getGoalWords(one.settings, today) : ''),
      wrong: one.wrong,
      says: isOver ? 'over budget' : percent(done),
      left: one.paused ? '' : getLeftWords(one),
      isOver,
      canStart: !one.paused && one.cards > 0,
    }
  })

/**
 * Whether a preset schedules anything at all: a deck points at it, and those
 * decks hold cards. A preset that schedules nothing today is still one of
 * these, and says why.
 */
const isScheduling = (one: Preset): boolean => one.named > 0 && one.faces > 0
