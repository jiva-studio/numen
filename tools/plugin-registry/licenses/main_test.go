package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const mitText = `MIT License

Copyright (c) 2026 Example Author

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
`

func writeFile(t *testing.T, dir, name, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestListLicensesReadsTheLicenseFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "LICENSE", mitText)
	writeFile(t, dir, "README.md", "# Example\n")

	ids, err := listLicenses(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"MIT"}) {
		t.Fatalf("ids = %v, want [MIT]", ids)
	}
}

func TestListLicensesIgnoresTextThatIsNotALicense(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "LICENSE.md", "All rights reserved by the author.\n")

	ids, err := listLicenses(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("ids = %v, want none", ids)
	}
}

func TestListLicensesOfAFolderWithNoLicenseFileIsEmpty(t *testing.T) {
	ids, err := listLicenses(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("ids = %v, want none", ids)
	}
}
