/** The size of a text field, which the shape a new note is dragged out as is drawn at. */
import { describe, expect, it } from 'vitest'
import { effectScope } from 'vue'
import { DESIGNED_FIELD, useFieldSize } from './fieldSize'

describe('the size of a field', () => {
  it('is the size the field was designed at where nothing is measured', () => {
    const scope = effectScope()
    const size = scope.run(() => useFieldSize())!
    expect(size.value).toStrictEqual(DESIGNED_FIELD)
    scope.stop()
  })
})
