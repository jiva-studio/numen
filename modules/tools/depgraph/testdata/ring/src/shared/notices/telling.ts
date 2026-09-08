/**
 * Two folders that each reach the other. Neither file is in a cycle, so the
 * file-level ring rule says nothing and the folder-level one refuses it.
 */
import { mark } from '../tabs/marking'

export const said = () => `going${mark()}`
