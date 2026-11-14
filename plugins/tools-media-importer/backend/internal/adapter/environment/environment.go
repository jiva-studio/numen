// Package environment reads the variables that select the mock downloader.
package environment

import "os"

// MockModelVariable names the variable that has the plugin reach video sites through the mock downloader.
const MockModelVariable = "NUMEN_TEST_MOCK_ML"

const trueValue = "true"

// IsMockModel returns whether the variables of the process ask for the mock downloader.
func IsMockModel() bool {
	return os.Getenv(MockModelVariable) == trueValue
}

// IsMockModelIn returns whether vars ask for the mock downloader.
func IsMockModelIn(vars map[string]string) bool {
	return vars[MockModelVariable] == trueValue
}
