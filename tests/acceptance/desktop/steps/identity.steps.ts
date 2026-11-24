import { Given, Then, When } from '../support/bdd.js'

When('the person opens the page {string}', async ({ person }, name: string) => {
  await person.settings.showPage(name)
})

When('the person starts a new identity', async ({ person }) => {
  await person.identity.start()
})

Then('the identity shows {int} recovery words', async ({ person }, count: number) => {
  await person.identity.expectWordCount(count)
})

Then('the identity does not show the recovery words', async ({ person }) => {
  await person.identity.expectNoWords()
})

When('the person writes the recovery words down', async ({ person }) => {
  await person.identity.writeWordsDown()
})

Then('the identity asks for {int} of the recovery words', async ({ person }, count: number) => {
  await person.identity.expectWordsAsked(count)
})

When('the person confirms {int} of the recovery words', async ({ person }, count: number) => {
  await person.identity.confirmWords(count)
})

When(
  'the person confirms the recovery words with the word {string} in place of one word',
  async ({ person }, word: string) => {
    await person.identity.confirmWithMistake(word)
  },
)

Given('the person creates an identity', async ({ person }) => {
  await person.identity.create()
})

Then('this device has {presence} identity', async ({ person }, has: boolean) => {
  await person.identity.expectIdentity(has)
})

When('the person deletes the identity from this device', async ({ person }) => {
  await person.identity.delete()
})

When('the person restores the identity from its recovery words', async ({ person }) => {
  await person.identity.restoreFromRecoveryWords()
})

When(
  'the person restores an identity from the word {string} written {int} times',
  async ({ person }, word: string, count: number) => {
    await person.identity.restoreFromWords(Array.from({ length: count }, () => word).join(' '))
  },
)

Then('the account fingerprint is the same as before', async ({ person }) => {
  await person.identity.expectSameAccount()
})

Then('the identity warns about {string}', async ({ person }, text: string) => {
  await person.identity.expectWarning(text)
})
