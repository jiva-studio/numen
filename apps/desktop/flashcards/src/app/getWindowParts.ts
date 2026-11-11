import { createWindow, type WindowDeps } from './window/createWindow'

/** The parts of the window, made once for the component that draws it. */
export function getWindowParts(deps?: WindowDeps) {
  return createWindow(deps)
}
