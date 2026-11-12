/** The person running a command that asks for arguments, over nothing, a file or folder, or a tab. */
import type { ApplicationDriver } from './application.js'
import type { FileTreeDriver } from './file-tree.js'
import type { PaletteDriver } from './palette.js'
import { parseArguments, type CommandTarget } from './rules.js'
import type { Scenario } from './scenario.js'
import type { TabDriver } from './tabs.js'

/** The drivers a command runs through. */
export interface CommandDrivers {
  readonly application: ApplicationDriver
  readonly tab: TabDriver
  readonly fileTree: FileTreeDriver
  readonly palette: PaletteDriver
}

/** The row that names the folder of a vault, which the native dialog of the command Create vault asks for. */
const FOLDER_ROW = 'Folder'

export class CommandDriver {
  constructor(
    private readonly scenario: Scenario,
    private readonly drivers: CommandDrivers,
  ) {}

  async run(label: string, target: CommandTarget, table: readonly (readonly string[])[]): Promise<void> {
    this.scenario.noteAction()
    const rows = parseArguments(table)
    const folders = rows.filter(([key]) => key === FOLDER_ROW)
    const asked = rows.filter(([key]) => key !== FOLDER_ROW)
    if (folders.length > 1) throw new Error(`the table has more than one row "${FOLDER_ROW}"`)
    if (folders.length === 1) this.chooseFolder(folders[0][1])
    await this.start(label, target)
    await this.drivers.palette.answer(asked)
  }

  /** Answers what a command that is already running asks for. */
  async answer(table: readonly (readonly string[])[]): Promise<void> {
    await this.drivers.palette.answer(parseArguments(table))
  }

  private chooseFolder(vault: string): void {
    const folder = this.scenario.getWorld().getFolder(this.scenario.getVault(vault))
    this.scenario.getServer().writeFolderChoice(folder)
  }

  private async start(label: string, target: CommandTarget): Promise<void> {
    if (target.kind === 'entry') return this.drivers.fileTree.runCommandOn(label, target.entry, target.path)
    if (target.kind === 'tab') await this.drivers.tab.activate(target.tab, target.reference, target.name)
    await this.drivers.application.runCommand(label)
  }
}
