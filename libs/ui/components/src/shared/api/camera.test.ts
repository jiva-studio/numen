/** The camera helpers say what a person can do and why a camera did not open. */
import { describe, expect, it } from 'vitest'
import { describeCameraFailure } from './camera'

describe('describeCameraFailure', () => {
  it('says the camera is not allowed', () => {
    expect(describeCameraFailure(new DOMException('no', 'NotAllowedError'))).toBe(
      'The camera is not allowed.',
    )
  })

  it('says no camera was found', () => {
    expect(describeCameraFailure(new DOMException('none', 'NotFoundError'))).toBe(
      'No camera was found.',
    )
  })

  it('says the camera did not open for any other cause', () => {
    expect(describeCameraFailure(new Error('busy'))).toBe('The camera did not open.')
  })
})
