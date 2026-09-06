import { describe, expect, it } from 'vitest'
import { Goal as Goals } from '@numen/protocol'

import { goalNames, goalOf } from './goal'

/** Every goal the schema carries, without the names its reverse mapping holds. */
const carried = Object.values(Goals).filter((one) => typeof one === 'number')

describe('the goal of a preset', () => {
  it('has a word for every goal but the unspecified one', () => {
    const worded = carried.filter((one) => goalOf[one] !== null)
    expect(worded).toStrictEqual(carried.filter((one) => one !== Goals.UNSPECIFIED))
  })

  it('sends back the value each word came from', () => {
    for (const one of carried) {
      const word = goalOf[one]
      if (word) expect(goalNames[word]).toBe(one)
    }
  })
})
