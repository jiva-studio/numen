/** Runs once before the scenarios: a run that must have the real sync server stops here, in plain words, when the environment names none. */
import { isStackRequired, NO_STACK_MESSAGE, readStack } from '../drivers/sync-stack.js'

export default function checkSyncStack(): void {
  if (readStack() === undefined && isStackRequired()) throw new Error(NO_STACK_MESSAGE)
}
