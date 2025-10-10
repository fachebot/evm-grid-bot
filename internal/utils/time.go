package utils

import "time"

var (
	DefaultTimeZone = 8
)

func FormaDate(t time.Time, timeZone ...int) string {
	zone := DefaultTimeZone
	if len(timeZone) > 0 {
		zone = timeZone[0]
	}
	loc := time.FixedZone("CST", 60*60*zone)
	return t.In(loc).Format("01/02 15:04:05")
}

func FormaTime(t time.Time, timeZone ...int) string {
	zone := DefaultTimeZone
	if len(timeZone) > 0 {
		zone = timeZone[0]
	}
	loc := time.FixedZone("CST", 60*60*zone)
	return t.In(loc).Format("2006/01/02 15:04:05")
}

func ParseTime(date string, timeZone ...int) (time.Time, error) {
	zone := DefaultTimeZone
	if len(timeZone) > 0 {
		zone = timeZone[0]
	}
	loc := time.FixedZone("CST", 60*60*zone)
	return time.ParseInLocation("2006/01/02 15:04:05", date, loc)
}
