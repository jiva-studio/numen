import type { DataTable } from 'playwright-bdd'
import { Then, When } from '../support/bdd.js'

When('the person clicks the button {string} in the {bar}', async ({ person }, label: string, _bar: string) => {
  await person.window.clickButton(label)
})

When(
  'the person clicks the button {string} in the title bar with:',
  async ({ person }, label: string, answers: DataTable) => {
    await person.window.clickButton(label)
    await person.command.answer(answers.raw())
  },
)

Then('the title bar has the button {string}', async ({ person }, label: string) => {
  await person.window.expectButton(label)
})

Then('the title bar does not have the button {string}', async ({ person }, label: string) => {
  await person.window.expectNoButton(label)
})

Then('the title bar has the update {string}', async ({ person }, version: string) => {
  await person.window.expectUpdate(version.replace(/^v/, ''))
})

Then('the title bar does not have an update', async ({ person }) => {
  await person.window.expectNoUpdate()
})

When('the person opens the update notice', async ({ person }) => {
  await person.window.openUpdateNotice()
})

Then('the update notice has the button {string}', async ({ person }, label: string) => {
  await person.window.expectNoticeButton(label)
})

Then('the window has the status bar', async ({ person }) => {
  await person.window.expectStatusBar()
})

Then('the window does not have the status bar', async ({ person }) => {
  await person.window.expectNoStatusBar()
})

When('the person opens the task list', async ({ person }) => {
  await person.window.openTaskList()
})

Then('the task list has the task {string}', async ({ person }, text: string) => {
  await person.window.expectTask(text)
})

Then('the task list does not have the task {string}', async ({ person }, text: string) => {
  await person.window.expectNoTask(text)
})

Then('the task list has the failed task {string}', async ({ person }, text: string) => {
  await person.window.expectFailedTask(text)
})

When('the person cancels the task {string}', async ({ person }, text: string) => {
  await person.window.cancelTask(text)
})

Then('the window shows the message {string}', async ({ person }, text: string) => {
  await person.window.expectMessage(text)
})

Then('the window has the following tabs:', async ({ person }, tabs: DataTable) => {
  await person.tab.expectTabs(tabs.raw())
})

When('the person answers {string} to {string}', async ({ person }, button: string, part: string) => {
  await person.window.answerQuestion(button, part)
})

When('the person answers {string} to the question:', async ({ person }, button: string, question: string) => {
  await person.window.answerWholeQuestion(button, question)
})

Then('the dialog does not show {string}', async ({ person }, text: string) => {
  await person.window.expectDialogWithout(text)
})
