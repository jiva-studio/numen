import { defineParameterType } from 'playwright-bdd'

defineParameterType({
  name: 'actor',
  regexp: /the person|another program/,
  transformer: (actor: string) => actor,
})

defineParameterType({
  name: 'has',
  regexp: /has|does not have/,
  transformer: (word: string) => word === 'has',
})
