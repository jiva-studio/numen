package completion

import "testing"

func TestGetModelTitleSeparatesProviderAndModelWithAMiddleDot(t *testing.T) {
	cases := map[string]string{
		"OpenRouter/DeepSeek R1 0528": "OpenRouter · DeepSeek R1 0528",
		"Sonnet 5.5":                  "Sonnet 5.5",
		"/model":                      "/model",
		"provider/":                   "provider/",
	}
	for title, want := range cases {
		if got := getModelTitle(title); got != want {
			t.Errorf("getModelTitle(%q) = %q, want %q", title, got, want)
		}
	}
}
