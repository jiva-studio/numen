/** The value held when the one received is equal to it, so what reads it sees no change. */
export const keepEqual = <T>(held: T | undefined, received: T): T =>
  held !== undefined && JSON.stringify(held) === JSON.stringify(received) ? held : received
