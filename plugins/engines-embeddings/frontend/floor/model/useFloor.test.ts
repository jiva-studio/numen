import { describe, expect, it } from 'vitest'
import { CALIBRATION_OUTCOME, FLOOR_SOURCE } from './codes'
import { createFakeClient, createState } from './fixtures'
import { useFloor } from './useFloor'

/** A Wait the test releases by hand. */
function createGate() {
  const pendingResolvers: Array<() => void> = []
  return {
    wait: () => new Promise<void>((resolve) => pendingResolvers.push(resolve)),
    release: () => pendingResolvers.splice(0).forEach((resolve) => resolve()),
    count: () => pendingResolvers.length,
  }
}

const flush = () => new Promise((resolve) => setTimeout(resolve, 0))

describe('useFloor', () => {
  it('loads the state', async () => {
    const client = createFakeClient()
    const floor = useFloor({ client })
    expect(floor.status.value).toBe('loading')
    await floor.load()
    expect(floor.status.value).toBe('ready')
    expect(floor.state.value?.floor).toBe(0.41)
    expect(client.getFloor).toHaveBeenCalledTimes(1)
  })

  it('keeps an unavailable embedder as an error state with its message', async () => {
    const client = createFakeClient()
    client.getFloor.mockRejectedValueOnce(new Error('[unavailable] embedder is not running'))
    const floor = useFloor({ client })
    await floor.load()
    expect(floor.status.value).toBe('error')
    expect(floor.loadError.value).toContain('unavailable')
    await floor.load()
    expect(floor.status.value).toBe('ready')
    expect(floor.loadError.value).toBe('')
  })

  it('keeps the shown state when a later load fails', async () => {
    const client = createFakeClient()
    const floor = useFloor({ client })
    await floor.load()
    client.getFloor.mockRejectedValueOnce(new Error('boom'))
    await floor.load()
    expect(floor.status.value).toBe('error')
    expect(floor.state.value).not.toBeNull()
  })

  it('calibrates, shows the busy state, and takes the state of the answer', async () => {
    const client = createFakeClient()
    const floor = useFloor({ client })
    await floor.load()
    client.calibrateFloor.mockImplementationOnce(async () => {
      expect(floor.isCalibrating.value).toBe(true)
      return {
        outcome: CALIBRATION_OUTCOME.DONE,
        state: createState({ floor: 0.5, automaticFloor: 0.5 }),
      }
    })
    await floor.calibrate()
    expect(floor.isCalibrating.value).toBe(false)
    expect(floor.state.value?.floor).toBe(0.5)
    expect(floor.isTooFew.value).toBe(false)
  })

  it('reads the state again when the calibration answer carries none', async () => {
    const client = createFakeClient()
    client.calibrateFloor.mockResolvedValueOnce({ outcome: CALIBRATION_OUTCOME.DONE })
    const floor = useFloor({ client })
    await floor.load()
    await floor.calibrate()
    expect(client.getFloor).toHaveBeenCalledTimes(2)
  })

  it('maps the TOO_FEW outcome to the not-enough state', async () => {
    const client = createFakeClient(
      createState({ calibration: undefined, source: FLOOR_SOURCE.MODEL }),
    )
    client.calibrateFloor.mockResolvedValueOnce({ outcome: CALIBRATION_OUTCOME.TOO_FEW })
    const floor = useFloor({ client })
    await floor.load()
    await floor.calibrate()
    expect(floor.isTooFew.value).toBe(true)
    expect(floor.actionError.value).toBe('')
  })

  it('shows a calibration error inline and leaves the busy state', async () => {
    const client = createFakeClient()
    client.calibrateFloor.mockRejectedValueOnce(new Error('[unavailable] no embedder'))
    const floor = useFloor({ client })
    await floor.load()
    await floor.calibrate()
    expect(floor.actionError.value).toContain('no embedder')
    expect(floor.isCalibrating.value).toBe(false)
  })

  it('ignores a second calibration while one runs', async () => {
    const client = createFakeClient()
    const gate = createGate()
    client.calibrateFloor.mockImplementationOnce(async () => {
      await gate.wait()
      return { outcome: CALIBRATION_OUTCOME.DONE }
    })
    const floor = useFloor({ client })
    await floor.load()
    const first = floor.calibrate()
    await floor.calibrate()
    gate.release()
    await first
    expect(client.calibrateFloor).toHaveBeenCalledTimes(1)
  })

  it('switches to the manual mode with the automatic value as the start', async () => {
    const client = createFakeClient()
    const floor = useFloor({ client })
    await floor.load()
    await floor.setMode(true)
    expect(client.setFloor).toHaveBeenCalledWith({ isManual: true, floor: 0.41 })
    expect(floor.isManual.value).toBe(true)
  })

  it('switches back to the automatic mode', async () => {
    const client = createFakeClient(createState({ isManual: true, manualFloor: 0.5, floor: 0.5 }))
    const floor = useFloor({ client })
    await floor.load()
    await floor.setMode(false)
    expect(client.setFloor).toHaveBeenCalledWith({ isManual: false, floor: 0.5 })
    expect(floor.isManual.value).toBe(false)
  })

  it('does not change the mode while the settings file sets the threshold', async () => {
    const client = createFakeClient(createState({ source: FLOOR_SOURCE.FILE }))
    const floor = useFloor({ client })
    await floor.load()
    await floor.setMode(true)
    await floor.setManualFloor(0.6)
    expect(client.setFloor).not.toHaveBeenCalled()
    expect(floor.isFileSet.value).toBe(true)
  })

  it('writes only the last slider value after the pause', async () => {
    const client = createFakeClient(createState({ isManual: true, manualFloor: 0.4, floor: 0.4 }))
    const gate = createGate()
    const floor = useFloor({ client, wait: gate.wait })
    await floor.load()
    const first = floor.setManualFloor(0.5)
    const second = floor.setManualFloor(0.6)
    expect(floor.manualValue.value).toBe(0.6)
    expect(client.setFloor).not.toHaveBeenCalled()
    gate.release()
    await Promise.all([first, second])
    expect(client.setFloor).toHaveBeenCalledTimes(1)
    expect(client.setFloor).toHaveBeenCalledWith({ isManual: true, floor: 0.6 })
  })

  it('waits the configured pause', async () => {
    const client = createFakeClient(createState({ isManual: true, manualFloor: 0.4, floor: 0.4 }))
    const waits: number[] = []
    const floor = useFloor({ client, wait: async (ms) => void waits.push(ms) })
    await floor.load()
    await floor.setManualFloor(0.5)
    expect(waits).toEqual([300])
  })

  it('drops a pending slider write when the mode changes', async () => {
    const client = createFakeClient(createState({ isManual: true, manualFloor: 0.4, floor: 0.4 }))
    const gate = createGate()
    const floor = useFloor({ client, wait: gate.wait })
    await floor.load()
    const pending = floor.setManualFloor(0.7)
    await floor.setMode(false)
    gate.release()
    await pending
    expect(client.setFloor).toHaveBeenCalledTimes(1)
    expect(client.setFloor).toHaveBeenCalledWith({ isManual: false, floor: 0.7 })
  })

  it('shows a write error inline', async () => {
    const client = createFakeClient()
    client.setFloor.mockRejectedValueOnce(new Error('invalid floor'))
    const floor = useFloor({ client })
    await floor.load()
    await floor.setMode(true)
    expect(floor.actionError.value).toBe('invalid floor')
  })

  it('checks a query', async () => {
    const client = createFakeClient()
    const floor = useFloor({ client })
    await floor.load()
    await floor.probe('  love  ')
    expect(client.probeFloor).toHaveBeenCalledWith({ query: 'love' })
    expect(floor.probeResult.value?.doesPass).toBe(true)
    expect(floor.isProbing.value).toBe(false)
  })

  it('skips an empty query', async () => {
    const client = createFakeClient()
    const floor = useFloor({ client })
    await floor.probe('   ')
    expect(client.probeFloor).not.toHaveBeenCalled()
  })

  it('shows a probe error and clears the earlier result', async () => {
    const client = createFakeClient()
    const floor = useFloor({ client })
    await floor.probe('love')
    client.probeFloor.mockRejectedValueOnce(new Error('[unavailable] no embedder'))
    await floor.probe('peace')
    expect(floor.probeResult.value).toBeNull()
    expect(floor.probeError.value).toContain('unavailable')
  })

  it('ignores answers that arrive after the scope ended', async () => {
    const { effectScope } = await import('vue')
    const client = createFakeClient()
    const gate = createGate()
    client.getFloor.mockImplementationOnce(async () => {
      await gate.wait()
      return createState()
    })
    const scope = effectScope()
    const floor = scope.run(() => useFloor({ client }))!
    const loading = floor.load()
    scope.stop()
    gate.release()
    await loading
    await flush()
    expect(floor.state.value).toBeNull()
  })
})
