import { Given, Then, When } from '../support/bdd.js'

Given('the tab {string} is open at {word} {string}', async ({ person }, file: string, place: string, value: string) => {
  await person.openTabAt(file, place, value)
})

When('the window starts again', async ({ person }) => {
  await person.startAgain()
})

Then('the tab {string} is at {word} {string}', async ({ person }, file: string, place: string, value: string) => {
  await person.expectTabAt(file, place, value)
})
