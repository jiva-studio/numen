/**
 * The order of the reads and writes of one remote store: a read is taken only where no write
 * overlapped it and no newer answer was taken.
 */

/** One read in flight: its place in the order of reads, and the writes when it began. */
export interface PendingRead {
  readonly place: number
  readonly settledWrites: number
  readonly isWriting: boolean
}

export type WriteOrder = ReturnType<typeof createWriteOrder>

export function createWriteOrder() {
  let writesInFlight = 0
  let settledWrites = 0
  let lastRead = 0
  let lastTaken = 0

  /** A write begins; the function returned settles it, whatever its answer. */
  const beginWrite = (): (() => void) => {
    writesInFlight += 1
    let isSettled = false
    return () => {
      if (isSettled) return
      isSettled = true
      writesInFlight -= 1
      settledWrites += 1
    }
  }

  const isWriteInFlight = (): boolean => writesInFlight > 0

  /** The number of writes settled so far. */
  const getSettledWrites = (): number => settledWrites

  const beginRead = (): PendingRead => ({
    place: ++lastRead,
    settledWrites,
    isWriting: writesInFlight > 0,
  })

  /** Whether no write was in flight while the read ran: its answer cannot predate a write. */
  const isReadAfterWrites = (pendingRead: PendingRead): boolean =>
    !pendingRead.isWriting && writesInFlight === 0 && pendingRead.settledWrites === settledWrites

  /** Whether the answer of the read is newer than every answer taken; a newer one is taken. */
  const takeRead = (pendingRead: PendingRead): boolean => {
    if (pendingRead.place < lastTaken) return false
    lastTaken = pendingRead.place
    return true
  }

  return { beginWrite, isWriteInFlight, getSettledWrites, beginRead, isReadAfterWrites, takeRead }
}
