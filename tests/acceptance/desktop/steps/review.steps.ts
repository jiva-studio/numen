import { Given, Then, When } from '../support/bdd.js'

Given('the person chooses the vault {string}', async ({ person }, vault: string) => {
  await person.review.chooseVault(vault)
})

When('the person starts the review of the {scope} {string}', async ({ person }, scope: string, name: string) => {
  await person.review.startReview(scope, name)
})

When('the person shows the answer of the card', async ({ person }) => {
  await person.review.showAnswer()
})

When('the person answers the card with {string}', async ({ person }, answer: string) => {
  await person.review.answer(answer)
})

Then('the card has the answer {string}', async ({ person }, answer: string) => {
  await person.review.expectAnswer(answer)
})

Then('the review session has {string}', async ({ person }, text: string) => {
  await person.review.expectSessionHas(text)
})
