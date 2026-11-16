import { createNodeIdFactory } from '../lib/identity'
import { browserRandomness } from './browserRandomness'

/** Node identities drawn from the window's randomness. */
export const createBrowserNodeIdFactory = () => createNodeIdFactory(browserRandomness)
