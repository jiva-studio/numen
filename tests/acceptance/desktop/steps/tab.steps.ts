import type { Place, TabKindChoice, TabRef } from '../drivers/rules.js'
import { Given, Then, When } from '../support/bdd.js'

When('the person closes the {tab} {reference} {string}', async ({ person }, kind: TabKindChoice, reference: string, name: string) => {
  await person.tab.close(kind, reference, name)
})

Then('the {tab} {reference} {string} is active', async ({ person }, kind: TabKindChoice, reference: string, name: string) => {
  await person.tab.expectActive(kind, reference, name)
})

Given(
  'the {tab} {reference} {string} is open at {place}',
  async ({ person }, kind: TabKindChoice, reference: string, name: string, place: Place) => {
    await person.tab.openAt(kind, reference, name, place)
  },
)

Then(
  'the {tab} {reference} {string} is at {place}',
  async ({ person }, kind: TabKindChoice, reference: string, name: string, place: Place) => {
    await person.tab.expectAt(kind, reference, name, place)
  },
)

Then('the {tab} {reference} {string} is open', async ({ person }, kind: TabKindChoice, reference: string, name: string) => {
  await person.tab.expectOpen(kind, reference, name)
})

Then('the {tab} {reference} {string} is not open', async ({ person }, kind: TabKindChoice, reference: string, name: string) => {
  await person.tab.expectNotOpen(kind, reference, name)
})

When(
  'the person runs the command {string} on the {tab} {reference} {string}',
  async ({ person }, label: string, kind: TabKindChoice, reference: string, name: string) => {
    await person.tab.activate(kind, reference, name)
    await person.application.runCommand(label)
  },
)

Then(
  'the {tab} {reference} {string} has {string}',
  async ({ person }, kind: TabKindChoice, reference: string, name: string, text: string) => {
    await person.tab.expectHasText(kind, reference, name, text)
  },
)

Then(
  'the {tab} {reference} {string} does not have {string}',
  async ({ person }, kind: TabKindChoice, reference: string, name: string, text: string) => {
    await person.tab.expectDoesNotHaveText(kind, reference, name, text)
  },
)

When('the person opens the file at {string}', async ({ person }, filePath: string) => {
  await person.tab.openFile(filePath)
})

When('the person plays the recording in the {tab} {reference} {string}', async ({ person }, kind: TabKindChoice, reference: string, name: string) => {
  await person.tab.playRecording(kind, reference, name)
})

Then('the recording in the {tab} {reference} {string} is playing', async ({ person }, kind: TabKindChoice, reference: string, name: string) => {
  await person.tab.expectPlaying(kind, reference, name)
})

Then(
  'the {tab} {reference} {string} shows a highlight',
  async ({ person }, kind: TabKindChoice, reference: string, name: string) => {
    await person.tab.expectHighlight(kind, reference, name)
  },
)

Then(
  'the {tab} {reference} {string} has the highlight {string}',
  async ({ person }, kind: TabKindChoice, reference: string, name: string, text: string) => {
    await person.tab.expectHighlight(kind, reference, name, text)
  },
)

When('the person opens the {tab} {reference} {string}', async ({ person }, kind: TabKindChoice, reference: string, name: string) => {
  await person.tab.activate(kind, reference, name)
})

When(
  'the person clicks the link {string} in the {tab} {reference} {string}',
  async ({ person }, link: string, kind: TabKindChoice, reference: string, name: string) => {
    await person.tab.clickLink(kind, reference, name, link)
  },
)

Then(
  'the {tab} {reference} {string} {possession} a recording',
  async ({ person }, kind: TabKindChoice, reference: string, name: string, has: boolean) => {
    await person.tab.expectHasRecording(kind, reference, name, has)
  },
)

When(
  'the person writes {string} in the field {string} of the card {string} of the {tabref}',
  async ({ person }, value: string, field: string, card: string, tab: TabRef) => {
    await person.deck.write(value, field, card, tab)
  },
)

Then(
  'the {tabref} has {string} in the field {string} of the card {string}',
  async ({ person }, tab: TabRef, value: string, field: string, card: string) => {
    await person.deck.expectValue(value, field, card, tab)
  },
)
