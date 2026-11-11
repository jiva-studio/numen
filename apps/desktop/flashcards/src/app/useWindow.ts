import { createWindow, type WindowDeps } from './window/createWindow'

/** The parts of the window, made once for the component that draws it. */
export function useWindow(deps?: WindowDeps) {
  return createWindow(deps)
}
