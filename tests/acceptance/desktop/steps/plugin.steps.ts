import { When } from '../support/bdd.js'

When('the person disables the plugin {string}', async ({ person }, name: string) => {
  await person.plugin.disable(name)
})
