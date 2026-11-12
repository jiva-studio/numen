import { Then, When } from '../support/bdd.js'

When('the person sends the message {string} to the agent', async ({ person }, message: string) => {
  await person.agent.send(message)
})

When('the person stops the agent', async ({ person }) => {
  await person.agent.stop()
})

Then('the composer of the agent is empty', async ({ person }) => {
  await person.agent.expectEmpty()
})

Then('the composer of the agent cannot send', async ({ person }) => {
  await person.agent.expectCannotSend()
})

Then('the composer of the agent has {string}', async ({ person }, text: string) => {
  await person.agent.expectText(text)
})

Then('the model of the agent is {string}', async ({ person }, model: string) => {
  await person.agent.expectModel(model)
})
