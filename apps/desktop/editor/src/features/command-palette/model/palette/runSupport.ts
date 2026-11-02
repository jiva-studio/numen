/**
 * The runs this build cannot perform, as reported to one window.
 *
 * The application reports an unsupported run the first time it is requested,
 * and the window does not offer it afterwards.
 */
import { shallowRef } from 'vue'
import type { RunSupport } from '../../types'

/** The runs supported by one window: all of them until the application reports one as unsupported. */
export const runSupport = (): RunSupport => {
  const completedRuns = shallowRef<ReadonlySet<string>>(new Set())
  return {
    canRun: (run) => !completedRuns.value.has(run),
    cannotRun: (run) => {
      completedRuns.value = new Set(completedRuns.value).add(run)
    },
  }
}
