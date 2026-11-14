package todo

import "time"

const dateLayout = "2006-01-02"

// IsDate reports whether text is a calendar date written as YYYY-MM-DD, leap years honored.
func IsDate(text string) bool {
	parsed, err := time.Parse(dateLayout, text)
	return err == nil && parsed.Format(dateLayout) == text
}

// AddDays returns the date that is count days after the date. It returns the date unchanged when
// the date is not a calendar date.
func AddDays(date string, count int) string {
	parsed, err := time.Parse(dateLayout, date)
	if err != nil {
		return date
	}
	return parsed.AddDate(0, 0, count).Format(dateLayout)
}
