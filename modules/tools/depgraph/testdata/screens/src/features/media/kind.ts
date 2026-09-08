/**
 * A screen filed under `features/`, reaching another one and reaching a screen
 * the window kept at its root. Both are a screen reaching a screen.
 */
import { view } from '../plex/view'
import { held } from '../../note/tab'
import { core } from '../../shared/core'
import { framed } from './url-tab/frame'

export const kind = () => `${view()} ${held()} ${core()} ${framed()}`
