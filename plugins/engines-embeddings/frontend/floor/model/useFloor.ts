/**
 * The state of the search relevance panel and the calls that change it.
 */
import { computed, getCurrentScope, onScopeDispose, ref, shallowRef } from 'vue'
import { CALIBRATION_OUTCOME, FLOOR_SOURCE } from './codes'
import { getErrorMessage, getInitialManual } from './present'
import type { FloorClient, FloorProbeResult, FloorState, Wait } from './types'
import { sleep } from './wait'

/** The pause after the last slider move before the value is written. */
export const SLIDER_DEBOUNCE_MS = 300

export interface UseFloorOptions {
  client: FloorClient
  wait?: Wait
  debounceMs?: number
}

export type FloorStatus = 'loading' | 'ready' | 'error'

export function useFloor(options: UseFloorOptions) {
  const { client } = options
  const wait = options.wait ?? sleep
  const debounceMs = options.debounceMs ?? SLIDER_DEBOUNCE_MS

  const state = shallowRef<FloorState | null>(null)
  const status = ref<FloorStatus>('loading')
  const loadError = ref('')
  const actionError = ref('')
  const isCalibrating = ref(false)
  const isTooFew = ref(false)
  const manualValue = ref(0)

  const probeResult = shallowRef<FloorProbeResult | null>(null)
  const probeError = ref('')
  const isProbing = ref(false)

  let generation = 0
  let isSliderPending = false
  let isDisposed = false
  if (getCurrentScope()) {
    onScopeDispose(() => {
      isDisposed = true
      generation += 1
    })
  }

  const isFileSet = computed(() => state.value?.source === FLOOR_SOURCE.FILE)
  const isManual = computed(() => state.value?.isManual === true)
  const isBusy = computed(() => isCalibrating.value)

  function applyState(next: FloorState | undefined): void {
    if (next === undefined || isDisposed) return
    state.value = next
    if (!isSliderPending) manualValue.value = getInitialManual(next)
    if (next.calibration !== undefined) isTooFew.value = false
  }

  async function load(): Promise<void> {
    status.value = state.value === null ? 'loading' : 'ready'
    try {
      applyState(await client.getFloor({}))
      loadError.value = ''
      status.value = 'ready'
    } catch (error) {
      if (isDisposed) return
      loadError.value = getErrorMessage(error)
      status.value = 'error'
    }
  }

  async function refresh(response: FloorState | undefined): Promise<void> {
    if (response !== undefined) {
      applyState(response)
      return
    }
    await load()
  }

  async function calibrate(): Promise<void> {
    if (isCalibrating.value) return
    isCalibrating.value = true
    actionError.value = ''
    try {
      const answer = await client.calibrateFloor({})
      isTooFew.value = answer.outcome === CALIBRATION_OUTCOME.TOO_FEW
      await refresh(answer.state)
    } catch (error) {
      if (!isDisposed) actionError.value = getErrorMessage(error)
    } finally {
      isCalibrating.value = false
    }
  }

  async function write(isManualMode: boolean, floor: number): Promise<void> {
    actionError.value = ''
    try {
      const answer = await client.setFloor({ isManual: isManualMode, floor })
      await refresh(answer.state)
    } catch (error) {
      if (!isDisposed) actionError.value = getErrorMessage(error)
    }
  }

  /** Switches between the automatic and the manual threshold, dropping a pending slider write. */
  async function setMode(isManualMode: boolean): Promise<void> {
    const current = state.value
    if (current === null || current.source === FLOOR_SOURCE.FILE) return
    generation += 1
    isSliderPending = false
    const start = isManualMode ? getInitialManual(current) : manualValue.value
    manualValue.value = start
    await write(isManualMode, start)
  }

  /** Moves the slider at once and writes the value after the pause that follows the last move. */
  async function setManualFloor(value: number): Promise<void> {
    if (state.value === null || isFileSet.value) return
    manualValue.value = value
    generation += 1
    isSliderPending = true
    const mine = generation
    await wait(debounceMs)
    if (mine !== generation || isDisposed) return
    isSliderPending = false
    await write(true, value)
  }

  async function probe(query: string): Promise<void> {
    const text = query.trim()
    if (text === '' || isProbing.value) return
    isProbing.value = true
    probeError.value = ''
    try {
      probeResult.value = await client.probeFloor({ query: text })
    } catch (error) {
      if (isDisposed) return
      probeResult.value = null
      probeError.value = getErrorMessage(error)
    } finally {
      isProbing.value = false
    }
  }

  /** Shows a failure that no action of this model handled itself. */
  function reportFailure(error: unknown): void {
    if (!isDisposed) actionError.value = getErrorMessage(error)
  }

  return {
    state,
    status,
    loadError,
    actionError,
    isCalibrating,
    isBusy,
    isTooFew,
    isFileSet,
    isManual,
    manualValue,
    probeResult,
    probeError,
    isProbing,
    load,
    calibrate,
    setMode,
    setManualFloor,
    probe,
    reportFailure,
  }
}

export type FloorModel = ReturnType<typeof useFloor>
