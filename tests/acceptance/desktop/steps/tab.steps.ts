import { Given, Then, When } from '../support/bdd.js'

When('the person closes the tab {reference} {string}', async ({ person }, reference: string, name: string) => {
  await person.tab.close(reference, name)
})

Then('the tab {reference} {string} is active', async ({ person }, reference: string, name: string) => {
  await person.tab.expectActive(reference, name)
})

Given(
  'the tab {reference} {string} is open at {position} {string}',
  async ({ person }, reference: string, name: string, position: string, where: string) => {
    await person.tab.openAt(reference, name, position, where)
  },
)

Then(
  'the tab {reference} {string} is at {position} {string}',
  async ({ person }, reference: string, name: string, position: string, where: string) => {
    await person.tab.expectAt(reference, name, position, where)
  },
)

Then('the tab {reference} {string} is open', async ({ person }, reference: string, name: string) => {
  await person.tab.expectOpen(reference, name)
})

Then(
  'the tab {reference} {string} has {string}',
  async ({ person }, reference: string, name: string, text: string) => {
    await person.tab.expectHasText(reference, name, text)
  },
)

Then(
  'the tab {reference} {string} does not have {string}',
  async ({ person }, reference: string, name: string, text: string) => {
    await person.tab.expectDoesNotHaveText(reference, name, text)
  },
)

When('the person opens the file at {string}', async ({ person }, filePath: string) => {
  await person.tab.openFile(filePath)
})

When('the person plays the recording in the tab {reference} {string}', async ({ person }, reference: string, name: string) => {
  await person.tab.playRecording(reference, name)
})

Then('the recording in the tab {reference} {string} is playing', async ({ person }, reference: string, name: string) => {
  await person.tab.expectPlaying(reference, name)
})

Then(
  'the tab {reference} {string} has the highlight {string}',
  async ({ person }, reference: string, name: string, text: string) => {
    await person.tab.expectHighlight(reference, name, text)
  },
)
