import type { DataTable } from 'playwright-bdd'
import { Then, When } from '../support/bdd.js'

When('the person sets the setting {string} to {string}', async ({ person }, name: string, value: string) => {
  await person.settings.set(name, value)
})

When(
  'the person sets the setting {string} of the plugin {string} to {string}',
  async ({ person }, name: string, plugin: string, value: string) => {
    await person.settings.set(name, value, plugin)
  },
)

When('the person clears the setting {string} of the plugin {string}', async ({ person }, name: string, plugin: string) => {
  await person.settings.set(name, '', plugin)
})

Then('the setting {string} is {string}', async ({ person }, name: string, value: string) => {
  await person.settings.expectValue(name, value)
})

Then(
  'the setting {string} of the plugin {string} is {string}',
  async ({ person }, name: string, plugin: string, value: string) => {
    await person.settings.expectValue(name, value, plugin)
  },
)

Then('the setting {string} does not have the option {string}', async ({ person }, name: string, option: string) => {
  await person.settings.expectOption(name, option, false)
})

Then(
  'the setting {string} of the plugin {string} has the option {string}',
  async ({ person }, name: string, plugin: string, option: string) => {
    await person.settings.expectOption(name, option, true, plugin)
  },
)

Then('the settings of the plugin {string} have the field {string}', async ({ person }, plugin: string, field: string) => {
  await person.settings.expectField(plugin, field, true)
})

Then('the settings of the plugin {string} do not have the field {string}', async ({ person }, plugin: string, field: string) => {
  await person.settings.expectField(plugin, field, false)
})

When(
  'the person sets the setting {string} of the plugin {string} to the following rows:',
  async ({ person }, name: string, plugin: string, rows: DataTable) => {
    await person.connections.setRows(name, plugin, rows.raw())
  },
)

Then(
  'the setting {string} of the plugin {string} has the following rows:',
  async ({ person }, name: string, plugin: string, rows: DataTable) => {
    await person.connections.expectRows(name, plugin, rows.raw())
  },
)

Then('the setting {string} of the plugin {string} has no rows', async ({ person }, name: string, plugin: string) => {
  await person.connections.expectRows(name, plugin, [])
})

When(
  'the person removes every row of the setting {string} of the plugin {string}',
  async ({ person }, name: string, plugin: string) => {
    await person.connections.setRows(name, plugin, [])
  },
)

Then(
  'the setting {string} of the plugin {string} shows {string}',
  async ({ person }, name: string, plugin: string, text: string) => {
    await person.connections.expectSettingText(name, plugin, text)
  },
)
