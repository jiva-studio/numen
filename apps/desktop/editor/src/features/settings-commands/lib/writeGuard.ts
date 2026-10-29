/**
 * The writes of one setting in flight. A read made while any is in flight is not
 * taken, and a write that fails goes back only when no newer choice was made.
 */
export function createWriteGuard() {
  let inFlight = 0
  let latest = 0

  /** Runs a write; the token indicates whether a newer choice was made since. */
  const run = async (token: number, send: () => Promise<string | null>) => {
    inFlight++
    try {
      return { failed: await send(), isLatest: token === latest }
    } finally {
      inFlight--
    }
  }

  return {
    /** A choice is made; its token is the newest. */
    beginChoice: () => ++latest,
    run,
    isIdle: () => inFlight === 0,
  }
}
