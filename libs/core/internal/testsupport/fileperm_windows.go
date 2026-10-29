package testsupport

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

// Shut makes a file that no user may open, and restores access when the test
// ends. On Windows that is an entry on the file denying every right to
// everyone.
func Shut(tb testing.TB, path string) {
	tb.Helper()

	everyone, err := windows.CreateWellKnownSid(windows.WinWorldSid)
	if err != nil {
		tb.Fatal(err)
	}
	refusing, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.GENERIC_ALL,
		AccessMode:        windows.DENY_ACCESS,
		Inheritance:       windows.NO_INHERITANCE,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_WELL_KNOWN_GROUP,
			TrusteeValue: windows.TrusteeValueFromSID(everyone),
		},
	}}, nil)
	if err != nil {
		tb.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, refusing, nil,
	); err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() { opening(tb, path) })

	// An account holding the backup privilege reads the file regardless of the
	// entry denying it, as root does where a mode denies access.
	if _, err := os.ReadFile(path); err == nil {
		tb.Skip("this account opens a file whatever the entry on it refuses")
	}
}

// opening gives the file back the rights its folder hands down, so what removes
// the folder may remove it.
func opening(tb testing.TB, path string) {
	if err := windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.UNPROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, nil, nil,
	); err != nil {
		tb.Error(err)
	}
}

// MakeUnwritable makes a file nothing may save over, and lets it be saved again
// alone: it reads as it always did, and a save renaming over the name is
func MakeUnwritable(tb testing.TB, path string) {
	tb.Helper()

	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		tb.Fatal(err)
	}
	handle, err := windows.CreateFile(
		name,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if err := windows.CloseHandle(handle); err != nil {
			tb.Error(err)
		}
	})
}
