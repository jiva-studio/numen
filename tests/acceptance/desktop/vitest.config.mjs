// The rules of the driver and the pure parts of the report are tested without a browser; the generated scenarios in .features-gen are Playwright's.
export default {
  test: { include: ['drivers/**/*.test.ts', 'report/**/*.test.ts'] },
}
