/**
 * Starts the window background processes and streams when the window mounts, and stops the start
 * when it unmounts.
 */
import { onMounted, onUnmounted, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import { waitWithinInterfaceLimit } from '@numen/plugin-loader'
import type { Caught } from '@/shared/api/errors'

export interface StartDeps {
  loadVaults(): Promise<void>
  startSettings(): Promise<void>
  startWindow(): Promise<void>
  startEditing(): void
  startLayout(): Promise<void>
  /** Starts every plugin; settles once all have started or failed. */
  startPlugins?(): Promise<void>
  /** Settles once the plugins of the window slots have started or failed. */
  waitForInterfacePlugins?(): Promise<void>
  /** Receives the failure of a start that is not awaited. */
  reportError(this: void, error: Caught): void
}

/** How long one of the steps the workspace depends on may take before the window says it could not load. */
const STEP_LIMIT_MS = 10_000

async function startPluginsWithinLimit(deps: StartDeps): Promise<void> {
  const starting = deps.startPlugins?.()
  if (!starting) return
  runInBackground(starting, deps.reportError)
  const interfaceStarted = deps.waitForInterfacePlugins?.() ?? starting
  await waitWithinInterfaceLimit(interfaceStarted)
}

async function withinLimit<T>(what: string, work: Promise<T>, limitMs: number): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined
  const late = new Promise<never>((_, reject) => {
    timer = setTimeout(
      () => reject(new Error(`${what} did not answer in ${limitMs / 1000} s`)),
      limitMs,
    )
  })
  /* Stryker disable BlockStatement,CallExpression: timer cleanup after Promise.race settles */
  try {
    return await Promise.race([work, late])
  } finally {
    clearTimeout(timer)
  }
  /* Stryker restore BlockStatement,CallExpression */
}

export function createAppStart(deps: StartDeps, stepLimitMs: number = STEP_LIMIT_MS) {
  /**
   * True once the plugins of the window slots have started, INTERFACE_WAIT_MS has passed since the window
   * mounted, or the start failed: the window is drawn from then on.
   */
  const isStarted = ref(false)
  /** Why the workspace did not load, shown with a retry; empty while it loads and once it has. */
  const failure = ref('')
  let attempt = 0
  const unmounting = new AbortController()

  const run = async (): Promise<void> => {
    // Stryker disable next-line UpdateOperator: generation counter only needs a distinct value per run
    const mine = ++attempt
    failure.value = ''
    try {
      await withinLimit('The vault list', deps.loadVaults(), stepLimitMs)
      await withinLimit('The settings request', deps.startSettings(), stepLimitMs)
      await startPluginsWithinLimit(deps)
      isStarted.value = true
      await withinLimit('The tab restore', deps.startLayout(), stepLimitMs)
    } catch (error) {
      isStarted.value = true
      if (mine === attempt) failure.value = error instanceof Error ? error.message : String(error)
      return
    }
    if (mine !== attempt) return
    runInBackground(deps.startWindow(), deps.reportError)
    deps.startEditing()
  }

  onMounted(() => {
    const running = run()
    runInBackground(
      waitWithinInterfaceLimit(running, unmounting.signal).then(() => {
        isStarted.value = true
      }),
      deps.reportError,
    )
    return running
  })

  onUnmounted(() => {
    unmounting.abort()
    // Stryker disable next-line UpdateOperator: generation counter only needs a distinct value per run
    attempt++
  })

  return { failure, isStarted, retry: run }
}
