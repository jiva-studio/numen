/** The person at the window as a whole: its keys, its title bar, its status bar, its list of background tasks and its dialogs. */
import { expect, type Locator } from 'playwright/test'
import { Question } from './components/Question.js'
import { StatusBar } from './components/StatusBar.js'
import { TaskList } from './components/TaskList.js'
import { TitleBar } from './components/TitleBar.js'
import { writeSetting } from '../../../e2e/desktop/tests/support/vault.js'
import { escapeRegExp, parseKeys } from './rules.js'
import type { Scenario } from './scenario.js'

const UPDATER = 'md.numen.tools.updater'
/** An update shows once the plugin has asked the feed, which a plugin that starts late does after this long. */
const UPDATE_LIMIT_MS = 30_000
/** A task shows once its plugin has started and reached the note, which the e2e test of the same task allows this long. */
const TASK_LIMIT_MS = 45_000

export class WindowDriver {
  private feedChecks = 0

  constructor(private readonly scenario: Scenario) {}

  private get titleBar(): TitleBar {
    return new TitleBar(this.scenario.page)
  }

  private get statusBar(): StatusBar {
    return new StatusBar(this.scenario.page)
  }

  private get question(): Question {
    return new Question(this.scenario.page)
  }

  private get taskList(): TaskList {
    return new TaskList(this.scenario.page)
  }

  /**
   * Makes the updater plugin ask the release feed again: the plugin asks when the window starts and when
   * its feed address changes, so a release published later is found by a new ending of the address.
   */
  async askFeedAgain(): Promise<void> {
    this.feedChecks += 1
    const address = `${this.scenario.getServer().getUpdateUrl()}?again=${this.feedChecks}`
    await writeSetting(this.scenario.page.request, [UPDATER, 'endpoint'], address)
  }

  async pressKeys(combination: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await this.scenario.page.keyboard.press(parseKeys(combination))
  }

  async clickButton(label: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    const button = this.titleBar.getButton(label)
    await expect(button, `the title bar has no button "${label}"`).toBeVisible()
    await button.click()
  }

  async expectButton(label: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await expect(this.titleBar.getButton(label)).toBeVisible()
  }

  async expectNoButton(label: string): Promise<void> {
    await expect(this.titleBar.getButton(label)).toHaveCount(0)
  }

  async expectUpdate(version: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await expect(this.titleBar.getUpdateButton()).toHaveText(`v${version}`, { timeout: UPDATE_LIMIT_MS })
  }

  async expectNoUpdate(): Promise<void> {
    await this.scenario.ensureWorkspace()
    await expect(this.titleBar.getUpdateButton()).toHaveCount(0)
  }

  async openUpdateNotice(): Promise<void> {
    await this.titleBar.getUpdateButton().click()
    await expect(this.titleBar.getUpdateNotice()).toBeVisible()
  }

  async expectNoticeButton(label: string): Promise<void> {
    await expect(this.titleBar.getNoticeButton(label)).toBeVisible()
  }

  async expectStatusBar(): Promise<void> {
    await this.scenario.ensureWorkspace()
    await expect(this.statusBar.root).toBeVisible()
  }

  async expectNoStatusBar(): Promise<void> {
    await this.scenario.ensureWorkspace()
    await expect(this.statusBar.root).not.toBeAttached()
  }

  /** Opens the list of background tasks, where the failed ones are listed, unless it is open. */
  async openTaskList(): Promise<Locator> {
    await this.scenario.ensureWorkspace()
    return this.taskList.open()
  }

  private async countTasks(text: string, isFailed: boolean): Promise<number> {
    await this.scenario.ensureWorkspace()
    return this.taskList.count(text, isFailed)
  }

  async expectTask(text: string): Promise<void> {
    await expect
      .poll(() => this.countTasks(text, false), { timeout: TASK_LIMIT_MS, message: `the task list has no task with the text "${text}"` })
      .toBeGreaterThan(0)
  }

  async expectNoTask(text: string): Promise<void> {
    await expect
      .poll(() => this.countTasks(text, false), { message: `the task list still has a task with the text "${text}"` })
      .toBe(0)
  }

  async expectFailedTask(text: string): Promise<void> {
    await expect
      .poll(() => this.countTasks(text, true), { message: `the task list has no failed task with the text "${text}"` })
      .toBeGreaterThan(0)
  }

  /** A message of the window is listed with the failed tasks of the status bar. */
  async expectMessage(text: string): Promise<void> {
    await this.expectFailedTask(text)
  }

  async cancelTask(text: string): Promise<void> {
    await this.expectTask(text)
    await this.taskList.cancel(text)
  }

  /** Whether a task of the list failed; the list stays as it was found. */
  async hasFailedTask(): Promise<boolean> {
    return this.taskList.hasFailed()
  }

  /** Answers the open question whose text contains the part. */
  async answerQuestion(button: string, part: string): Promise<void> {
    await this.question.answer(button, new RegExp(escapeRegExp(part)))
  }

  /** Answers the open question whose text is the text, line breaks and spaces aside. */
  async answerWholeQuestion(button: string, text: string): Promise<void> {
    await this.question.answer(button, text.replace(/\s+/g, ' ').trim())
  }

  /** The open dialog has the text neither in what it reads nor in a field. */
  async expectDialogWithout(text: string): Promise<void> {
    const dialog = this.scenario.page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await expect(dialog).not.toContainText(text)
    const values = await dialog.locator('input, textarea').evaluateAll((fields) => fields.map((field) => (field as HTMLInputElement).value))
    expect(values).not.toContain(text)
  }
}
