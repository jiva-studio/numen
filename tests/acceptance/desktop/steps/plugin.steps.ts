import { Then, When } from '../support/bdd.js'

When('the person disables the plugin {string}', async ({ person }, name: string) => {
  await person.plugin.disable(name)
})

When('the person enables the plugin {string}', async ({ person }, name: string) => {
  await person.plugin.enable(name)
})

Then('the plugin {string} is enabled', async ({ person }, name: string) => {
  await person.plugin.expectEnabled(name, true)
})

Then('the plugin {string} is disabled', async ({ person }, name: string) => {
  await person.plugin.expectEnabled(name, false)
})
