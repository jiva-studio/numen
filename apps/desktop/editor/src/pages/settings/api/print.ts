/** The print dialog of the window: the one edge the page asks the system to print through. */

/** Asks the system to print the page. */
export interface Printer {
  print(): void
}

/** The printer of this window. */
export const createPrinter = (): Printer => ({
  print: () => {
    window.print()
  },
})
