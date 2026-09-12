/**
 * The runs this build cannot do at all, as one window has been told them.
 *
 * The application says so the first time one is asked for, and that window
 * offers it nowhere after that.
 */
import { shallowRef } from 'vue'
import type { RunSupport } from '../types'

/** The runs one window holds, which is every one of them until it is told otherwise. */
export const runSupport = (): RunSupport => {
  const beyond = shallowRef<ReadonlySet<string>>(new Set())
  return {
    canRun: (run) => !beyond.value.has(run),
    cannotRun: (run) => void (beyond.value = new Set(beyond.value).add(run)),
  }
}
