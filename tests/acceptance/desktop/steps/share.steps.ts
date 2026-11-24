import { Given, Then, When } from '../support/bdd.js'
import type { ReaderName } from '../drivers/shared-reader.js'

Given('{reader} sets up an identity', async ({ person }, reader: ReaderName) => {
  await person.readers.createIdentity(reader)
})

When(
  'the person shares the folder {string} of the vault {string} with {reader}',
  async ({ person }, folder: string, _vault: string, reader: ReaderName) => {
    await person.sharing.shareFolder(folder, reader)
  },
)

Then(
  '{reader} has the note {string} with the text {string} in the shared vault {string}',
  async ({ person }, reader: ReaderName, title: string, text: string, vault: string) => {
    await person.readers.expectNote(reader, vault, title, text)
  },
)

Then(
  '{reader} does not have the file {string} in the shared vault {string}',
  async ({ person }, reader: ReaderName, filePath: string, vault: string) => {
    await person.readers.expectNoFile(reader, vault, filePath)
  },
)

Then('{reader} has no vault {string}', async ({ person }, reader: ReaderName, vault: string) => {
  await person.readers.expectNoVault(reader, vault)
})

Then('the sharing of the folder {string} lists the device of the reader with its account fingerprint', async ({ person }, folder: string) => {
  await person.sharing.expectReaderListed(folder)
})

When('the link is given to a second reader', async ({ person }) => {
  await person.sharing.giveLinkTo('second reader')
})

Then('{teller} is told {string}', async ({ person }, teller: string, text: string) => {
  if (teller === 'the person') await person.sharing.expectTold(text)
  else await person.readers.expectTold(teller === 'the reader' ? 'reader' : 'second reader', text)
})

Then(
  '{reader} is told {string} in the shared vault {string}',
  async ({ person }, reader: ReaderName, text: string, _vault: string) => {
    await person.readers.expectTold(reader, text)
  },
)

When('the person removes the access of the device of the reader to the folder {string}', async ({ person }, folder: string) => {
  await person.sharing.removeReader(folder)
})

When(
  '{reader} creates the note {string} in the shared vault {string}',
  async ({ person }, reader: ReaderName, title: string, vault: string) => {
    await person.readers.createNote(reader, vault, title)
  },
)

When(
  'the person shares the folder {string} of the vault {string} with {reader} and declines the request',
  async ({ person }, folder: string, _vault: string, reader: ReaderName) => {
    await person.sharing.declineFolder(folder, reader)
  },
)

When(
  '{reader} shares the shared vault {string} with the second reader',
  async ({ person }, reader: ReaderName, vault: string) => {
    await person.readers.shareVault(reader, vault)
  },
)

When(
  '{reader} adds the shared vault {string} to its synced vaults',
  async ({ person }, reader: ReaderName, vault: string) => {
    await person.readers.connectVault(reader, vault)
  },
)

Then('the subscription shows {string} on the reader', async ({ person }, text: string) => {
  await person.readers.expectSubscription('reader', text)
})

When('{reader} leaves the shared vault {string}', async ({ person }, reader: ReaderName, vault: string) => {
  await person.readers.leaveShare(reader, vault)
})

Then(
  'the folder of the shared vault {string} of {reader} still has the file {string} with the text {string}',
  async ({ person }, vault: string, reader: ReaderName, filePath: string, text: string) => {
    await person.readers.expectFolderKept(reader, vault, filePath, text)
  },
)
