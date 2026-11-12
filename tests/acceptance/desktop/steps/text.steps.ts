import type { TextKind } from '../drivers/rules.js'
import { When } from '../support/bdd.js'

When('the person recognizes the text of the file at {string}', async ({ person }, filePath: string) => {
  await person.text.recognize(filePath)
})

When('the person transcribes the recording at {string}', async ({ person }, filePath: string) => {
  await person.text.transcribe(filePath)
})

When('the person proofreads the {text} of the file at {string}', async ({ person }, kind: TextKind, filePath: string) => {
  await person.text.proofread(filePath, kind)
})

When('the person imports the link {string}', async ({ person }, address: string) => {
  await person.text.importLink(address)
})

When('the person downloads the text of the link {string}', async ({ person }, address: string) => {
  await person.text.downloadText(address)
})

When('the person downloads the video of the link {string}', async ({ person }, address: string) => {
  await person.text.downloadVideo(address)
})

When('the person opens the note of the link {string} in the file tree', async ({ person }, address: string) => {
  await person.text.openLink(address)
})

When('the person deletes the text of the link {string}', async ({ person }, address: string) => {
  await person.text.deleteText(address)
})

When('the person deletes the video of the link {string}', async ({ person }, address: string) => {
  await person.text.deleteVideo(address)
})
