import { When } from '../support/bdd.js'

When('the person opens the folder at {string}', async ({ person }, folder: string) => {
  await person.fileTree.open('folder', folder)
})

When('the person moves the file at {string} to the folder at {string}', async ({ person }, file: string, folder: string) => {
  await person.fileTree.move(file, folder)
})

When(
  'the person runs the command {string} on the {entry} at {string}',
  async ({ person }, label: string, entry: string, entryPath: string) => {
    await person.fileTree.runCommandOn(label, entry, entryPath)
  },
)
