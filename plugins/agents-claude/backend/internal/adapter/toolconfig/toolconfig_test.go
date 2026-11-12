package toolconfig

import "testing"

func TestWriteRequiresBothUrlAndToken(t *testing.T) {
	file, err := (Files{}).Write("http://127.0.0.1", "")
	if err != nil {
		t.Fatal(err)
	}
	if file != "" {
		t.Errorf("expected no config file when token is empty, got %q", file)
	}
	file2, err := (Files{}).Write("", "token")
	if err != nil {
		t.Fatal(err)
	}
	if file2 != "" {
		t.Errorf("expected no config file when url is empty, got %q", file2)
	}
}
