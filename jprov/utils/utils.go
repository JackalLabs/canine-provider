package utils

import "fmt"

const (
	UPTIME_LEFT_KEY = "UPTL-"
	FILE_KEY        = "FILE-"
	DOWNTIME_KEY    = "DWNT-"
)

func MakeUptimeKey(cid string) []byte {
	return []byte(fmt.Sprintf("%s%s", UPTIME_LEFT_KEY, cid))
}

func MakeFileKey(cid string) []byte {
	return []byte(fmt.Sprintf("%s%s", FILE_KEY, cid))
}

func MakeDowntimeKey(cid string) []byte {
	return []byte(fmt.Sprintf("%s%s", DOWNTIME_KEY, cid))
}
