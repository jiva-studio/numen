import { afterEach, describe, expect, it, vi } from 'vitest'
import { createPrinter } from './print'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('createPrinter', () => {
  it('opens the print dialog of the window once for each print', () => {
    const print = vi.spyOn(window, 'print').mockImplementation(() => undefined)
    const printer = createPrinter()
    printer.print()
    expect(print).toHaveBeenCalledOnce()
    printer.print()
    expect(print).toHaveBeenCalledTimes(2)
  })

  it('calls nothing until print is called', () => {
    const print = vi.spyOn(window, 'print').mockImplementation(() => undefined)
    createPrinter()
    expect(print).not.toHaveBeenCalled()
  })
})
