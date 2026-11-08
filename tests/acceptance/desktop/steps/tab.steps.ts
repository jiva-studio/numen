import { Given, Then, When } from '../support/bdd.js'

When('the person closes the tab {word} {string}', async ({ person }, how: string, value: string) => {
  await person.tab.close(how, value)
})

Then('the tab {word} {string} is active', async ({ person }, how: string, value: string) => {
  await person.tab.expectActive(how, value)
})

Given(
  'the tab {word} {string} is open at {word} {string}',
  async ({ person }, how: string, value: string, place: string, at: string) => {
    await person.tab.openAt(how, value, place, at)
  },
)

Then(
  'the tab {word} {string} is at {word} {string}',
  async ({ person }, how: string, value: string, place: string, at: string) => {
    await person.tab.expectAt(how, value, place, at)
  },
)

Then('the tab {word} {string} is open', async ({ person }, how: string, value: string) => {
  await person.tab.expectOpen(how, value)
})

Then(
  'the tab {word} {string} {has} {string}',
  async ({ person }, how: string, value: string, shows: boolean, text: string) => {
    await person.tab.expectShows(how, value, shows, text)
  },
)

When('the person opens the file at {string}', async ({ person }, filePath: string) => {
  await person.tab.openFile(filePath)
})

When('the person plays the recording in the tab {word} {string}', async ({ person }, how: string, value: string) => {
  await person.tab.playRecording(how, value)
})

Then('the recording in the tab {word} {string} is playing', async ({ person }, how: string, value: string) => {
  await person.tab.expectPlaying(how, value)
})
