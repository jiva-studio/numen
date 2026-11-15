import path from 'node:path'
import type {
  FullConfig,
  FullResult,
  Reporter,
  TestCase,
  TestResult,
} from 'playwright/types/testReporter'
import { type BddFile, readBddFile } from './bdd-data.js'
import { PLATFORM } from '../support/video.js'
import { getEnvironment } from './environment.js'
import { type Feature, readFeature } from './gherkin.js'
import { readGitRevision } from './overview.js'
import { createScenarioRecord, type ScenarioRecord } from './scenario.js'
import { clearFolder, writeReport } from './write-report.js'

type Options = { readonly outputFolder?: string }

export default class ScenarioReport implements Reporter {
  private readonly outputFolder: string
  private configDir = process.cwd()
  private rootDir = process.cwd()
  private readonly scenarios = new Map<string, ScenarioRecord>()
  private readonly bddFiles = new Map<string, BddFile>()
  private readonly features = new Map<string, Feature>()

  constructor(options: Options = {}) {
    this.outputFolder = options.outputFolder ?? 'acceptance-report'
  }

  get outputDir(): string {
    return path.resolve(this.configDir, this.outputFolder)
  }

  printsToStdio(): boolean {
    return false
  }

  onBegin(config: FullConfig): void {
    this.configDir = config.configFile ? path.dirname(config.configFile) : config.rootDir
    this.rootDir = path.resolve(this.configDir, '../../..')
    clearFolder(this.outputDir)
  }

  onTestEnd(test: TestCase, result: TestResult): void {
    if (!test.location.file.endsWith('.feature.spec.js')) return
    const bddFile = this.getBddFile(test.location.file)
    const record = createScenarioRecord(test, result, {
      bddFile,
      feature: this.getFeature(bddFile.featurePath),
      rootDir: this.rootDir,
      outputDir: this.outputDir,
      environment: getEnvironment(
        test.parent.project()?.name ?? 'chromium',
        result.annotations.find((one) => one.type === PLATFORM)?.description ?? process.platform,
      ),
    })
    if (record) this.scenarios.set(`${record.id} ${record.environment.id}`, record)
  }

  onEnd(result: FullResult): void {
    writeReport(this.outputDir, [...this.scenarios.values()], {
      startedAt: result.startTime,
      ...readGitRevision(this.rootDir),
    })
    process.stdout.write(`scenario report: ${path.join(this.outputDir, 'index.html')}\n`)
  }

  private getBddFile(specPath: string): BddFile {
    const known = this.bddFiles.get(specPath)
    if (known) return known
    const read = readBddFile(specPath, this.configDir)
    this.bddFiles.set(specPath, read)
    return read
  }

  private getFeature(featurePath: string): Feature {
    const known = this.features.get(featurePath)
    if (known) return known
    const read = readFeature(featurePath)
    this.features.set(featurePath, read)
    return read
  }
}
