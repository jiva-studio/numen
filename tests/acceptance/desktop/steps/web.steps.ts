import { Given } from '../support/bdd.js'

Given('the web address {string} has the page text {string}', ({ person }, address: string, text: string) => {
  person.web.setPageText(address, text)
})
