package settings

// Editor is the settings section of how editor text is drawn. The declaration of the shell gives
// each its type, its bounds and its default.
type Editor struct {
	FontSize    float64 `json:"fontSize"`
	FontFamily  string  `json:"fontFamily"`
	LineHeight  float64 `json:"lineHeight"`
	LineNumbers bool    `json:"lineNumbers"`
	SoftWrap    bool    `json:"softWrap"`
}

// DefaultEditor returns the editor settings the declaration of the shell gives.
func DefaultEditor() Editor {
	return Editor{
		FontSize:    getAppNumber(keyEditorFontSize),
		FontFamily:  getAppString(keyEditorFontFamily),
		LineHeight:  getAppNumber(keyEditorLineHeight),
		LineNumbers: getAppBool(keyEditorLineNumbers),
		SoftWrap:    getAppBool(keyEditorSoftWrap),
	}
}

func (e *Editor) reset(key string) {
	switch key {
	case keyEditorFontSize:
		e.FontSize = getAppNumber(key)
	case keyEditorFontFamily:
		e.FontFamily = getAppString(key)
	case keyEditorLineHeight:
		e.LineHeight = getAppNumber(key)
	case keyEditorLineNumbers:
		e.LineNumbers = getAppBool(key)
	case keyEditorSoftWrap:
		e.SoftWrap = getAppBool(key)
	}
}
