import { nextTick } from 'vue'
import { reportFailure, runInBackground } from '@numen/ui'

/** Runs the callback once the DOM reflects the state changed so far. */
export function runAfterRender(callback: () => void): void {
  runInBackground(nextTick(callback), reportFailure)
}
