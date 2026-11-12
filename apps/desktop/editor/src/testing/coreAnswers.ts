/**
 * Answers the calls of the core from a test, through the real clients, and takes the answers away
 * when the test is over.
 */
import { afterEach } from 'vitest'
import { installCoreAnswers, type CoreAnswers } from '@numen/core-client/testing'

const restores: (() => void)[] = []

afterEach(() => {
  for (const restore of restores.splice(0).reverse()) restore()
})

/** The core answers as `answers` say until the test ends; a call left out is refused. */
export const answerCore = (answers: CoreAnswers): void => {
  restores.push(installCoreAnswers(answers))
}
