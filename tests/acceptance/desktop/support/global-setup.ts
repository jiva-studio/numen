/** Runs once before the scenarios: they start from the saved test identities, so a stale copy stops the run here, in plain words. */
import { checkSeedIsCurrent } from '../drivers/identity-seed.js'

export default function checkIdentities(): void {
  checkSeedIsCurrent()
}
