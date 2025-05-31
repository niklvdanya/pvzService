package cli

import (
	"time"
)

const TimeFormat = "2006-01-02"

func MapStringToTime(dateStr string) (time.Time, error) {
	return time.Parse(TimeFormat, dateStr)
}

func MapTimeToString(t time.Time) string {
	return t.Format(TimeFormat)
}

func MapPackageType(packageType string) string {
	if packageType == "" {
		return "none"
	}
	return packageType
}
