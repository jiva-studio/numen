/** The only place the step keywords come from, so every step receives the `person` fixture. */
import { createBdd } from 'playwright-bdd'
import { test } from './test.js'

export const { Given, When, Then } = createBdd(test)
