/**
 * The window composed in four steps, each made from the ones before: the base (the tabs, the
 * vault, the commands of the features), the plugins, the palettes, and the start. Every call to the
 * core goes over the transports the window is given.
 */
import type { WindowTransports } from '@/shared/api/clients'
import { createWindowApi } from './createWindowApi'
import { createWindowBase } from './createWindowBase'
import { createWindowPalettes } from './createWindowPalettes'
import { createWindowPlugins } from './createWindowPlugins'
import { createWindowStart } from './createWindowStart'

/** Everything the window is made of, made once and handed to what draws it. */
export const createWindow = (transports: WindowTransports) => {
  const api = createWindowApi(transports)
  const base = createWindowBase(api)
  const { log, rest, frame, status, registry, connections, settingsRequests } = base
  const { tabs, tabParts, vault, windowSettings, getTarget, appCommands } = base
  const plugins = createWindowPlugins({
    api,
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
    api,
    log,
    rest,
    registry,
    settingsRequests,
    tabs,
    vault,
    windowSettings,
    getTarget,
    appCommands,
    plugins,
  })
  const { views, tabMenu, bootstrap } = createWindowStart({
    api,
    log,
    rest,
    frame,
    connections,
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
