package main

import (
	"testing"
)

const FLOATING_POINT_THRESHOLD = 0.000001

func TestParseOlderThanToHoursInvalidFormat(t *testing.T) {
	t.Parallel()

	_, err := parseOlderThanToHours("not-a-valid-format")
	if err == nil {
		t.Fatal("Expected to get an error when parsing an invalid format, but got nil")
	}
}

func TestParseOlderThanToHoursNegativeHours(t *testing.T) {
	t.Parallel()

	_, err := parseOlderThanToHours("-15h")
	if err == nil {
		t.Fatal("Expected to get an error when parsing a negative value, but got nil")
	}
}

func TestParseOlderThanToHoursZeroHours(t *testing.T) {
	t.Parallel()
	testParseOlderThan("0h", 0, t)
}

func TestParseOlderThanToHoursOneHour(t *testing.T) {
	t.Parallel()
	testParseOlderThan("1h", 1, t)
}

func TestParseOlderThanToHoursTenHours(t *testing.T) {
	t.Parallel()
	testParseOlderThan("10h", 10, t)
}

func TestParseOlderThanToHoursNineHundredNinetyNineHours(t *testing.T) {
	t.Parallel()
	testParseOlderThan("999h", 999, t)
}

func TestParseOlderThanToHoursZeroMinutes(t *testing.T) {
	t.Parallel()
	testParseOlderThan("0m", 0, t)
}

func TestParseOlderThanToHoursOneMinute(t *testing.T) {
	t.Parallel()
	testParseOlderThan("1m", 0.01666666666667, t)
}

func TestParseOlderThanToHoursTenMinutes(t *testing.T) {
	t.Parallel()
	testParseOlderThan("10m", 0.16666666666667, t)
}

func TestParseOlderThanToHoursSixtyMinutes(t *testing.T) {
	t.Parallel()
	testParseOlderThan("60m", 1, t)
}

func TestParseOlderThanToHoursNineHundredNinetyNineMinutes(t *testing.T) {
	t.Parallel()
	testParseOlderThan("999m", 16.65, t)
}

func TestParseOlderThanToHoursZeroDays(t *testing.T) {
	t.Parallel()
	testParseOlderThan("0d", 0, t)
}

func TestParseOlderThanToHoursOneDay(t *testing.T) {
	t.Parallel()
	testParseOlderThan("1d", 24, t)
}

func TestParseOlderThanToHoursTenDays(t *testing.T) {
	t.Parallel()
	testParseOlderThan("10d", 240, t)
}

func TestParseOlderThanToHoursNineHundredNinetyNineDays(t *testing.T) {
	t.Parallel()
	testParseOlderThan("999d", 23976, t)
}

func testParseOlderThan(timeFormat string, expectedHours float64, t *testing.T) {
	hours, err := parseOlderThanToHours(timeFormat)
	if err != nil {
		t.Fatalf("Unexpected error parsing a valid time format '%s': %s", timeFormat, err.Error())
	}

	diff := expectedHours - hours
	if diff > FLOATING_POINT_THRESHOLD {
		t.Fatalf("Expected %9f but got %9f. The difference %9f is greater than the floating point threshold %9f.", expectedHours, hours, diff, FLOATING_POINT_THRESHOLD)
	}
}