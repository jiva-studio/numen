/**
 * The window composed in four steps, each made from the ones before: the base (the tabs, the
 * vault, the commands of the features), the plugins, the palettes, and the start.
 */
import { createWindowBase } from './createWindowBase'
import { createWindowPalettes } from './createWindowPalettes'
import { createWindowPlugins } from './createWindowPlugins'
import { createWindowStart } from './createWindowStart'

/** Everything the window is made of, made once and handed to what draws it. */
export const createWindow = () => {
  const base = createWindowBase()
  const { log, rest, connections, frame, writeMessage, status, registry, settingsRequests } = base
  const { tabs, tabParts, vault, windowSettings, getTarget, appCommands } = base
  const plugins = createWindowPlugins({
    log,
    rest,
    frame,
    status,
    registry,
    tabs,
    vault,
    windowSettings,
    getTarget,
    appCommands,
  })
  const palettes = createWindowPalettes({
    rest,
    writeMessage,
    registry,
    openSettings: settingsRequests.emit,
    tabs,
    vault,
    windowSettings,
    getTarget,
    appCommands,
    plugins,
  })
  const { views, tabMenu, bootstrap } = createWindowStart({
    log,
    rest,
    connections,
    frame,
    writeMessage,
    settingsRequests,
    tabs,
    tabParts,
    vault,
    windowSettings,
    getTarget,
    plugins,
    palettes,
  })
  return { tabs, tabParts, vault, getTarget, palettes, views, tabMenu, bootstrap }
}

/** What the component that draws the window is given. */
export type WindowParts = ReturnType<typeof createWindow>
