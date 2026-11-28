/** The camera helpers say what a person can do and why a camera did not open. */
import { afterEach, describe, expect, it } from 'vitest'
import { describeCameraFailure, openMachineCamera } from './camera'

describe('describeCameraFailure', () => {
  it('says the camera is not allowed', () => {
    expect(describeCameraFailure(new DOMException('no', 'NotAllowedError'))).toBe(
      'The camera is not allowed.',
    )
  })

  it('says no camera was found', () => {
    expect(describeCameraFailure(new DOMException('none', 'NotFoundError'))).toBe(
      'No camera found.',
    )
  })

  it('says the camera did not open for any other cause', () => {
    expect(describeCameraFailure(new Error('busy'))).toBe('The camera did not open.')
  })
})

describe('openMachineCamera', () => {
  const stream = {} as MediaStream

  const useDevices = (getUserMedia: (constraints: MediaStreamConstraints) => Promise<MediaStream>) => {
    Object.defineProperty(navigator, 'mediaDevices', { value: { getUserMedia }, configurable: true })
  }

  afterEach(() => {
    Reflect.deleteProperty(navigator, 'mediaDevices')
  })

  it('opens the camera that faces away when the machine has one', async () => {
    const asked: MediaStreamConstraints[] = []
    useDevices(async (constraints) => {
      asked.push(constraints)
      return stream
    })
    expect(await openMachineCamera()).toBe(stream)
    expect(asked).toEqual([{ video: { facingMode: { ideal: 'environment' } } }])
  })

  it('opens any camera when the machine has only a camera that faces the person', async () => {
    const asked: MediaStreamConstraints[] = []
    useDevices(async (constraints) => {
      asked.push(constraints)
      if (asked.length === 1) throw new DOMException('no such camera', 'OverconstrainedError')
      return stream
    })
    expect(await openMachineCamera()).toBe(stream)
    expect(asked[1]).toEqual({ video: true })
  })

  it('does not ask again when the person or the system refused the camera', async () => {
    let calls = 0
    useDevices(async () => {
      calls += 1
      throw new DOMException('refused', 'NotAllowedError')
    })
    await expect(openMachineCamera()).rejects.toThrow('refused')
    expect(calls).toBe(1)
  })
})
