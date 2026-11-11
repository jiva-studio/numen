/** The text of a duration in seconds. */
export function getDuration(ms) {
  return (ms / 1000).toFixed(1) + "s";
}

const COLOR = process.stdout.isTTY ? { red: "\x1b[31m", green: "\x1b[32m", bold: "\x1b[1m", off: "\x1b[0m" } : { red: "", green: "", bold: "", off: "" };

/**
 * Prints the findings of the failed checks and one summary, and returns the exit code of the run.
 *
 * @param {Array<{ name: string, status: "pass" | "fail" | "skip", ms: number, output?: string }>} results
 */
export function printReport(results) {
  for (const one of results.filter((r) => r.status === "fail")) {
    console.log(`\n${COLOR.bold}${COLOR.red}[FAIL]${COLOR.off} ${one.name}`);
    if (one.output) console.log(one.output.trimEnd());
  }
  console.log(`\n${COLOR.bold}=== Summary ===${COLOR.off}`);
  for (const one of results) {
    const badge = one.status === "pass" ? `${COLOR.green}[PASS]${COLOR.off}` : one.status === "fail" ? `${COLOR.red}[FAIL]${COLOR.off}` : "[SKIP]";
    console.log(`  ${badge} ${one.name} (${getDuration(one.ms)})`);
  }
  const failed = results.filter((r) => r.status === "fail");
  console.log(failed.length === 0 ? `\n${results.length} checks, all passed.` : `\n${results.length} checks, ${failed.length} failed: ${failed.map((r) => r.name).join(", ")}`);
  return failed.length === 0 ? 0 : 1;
}
