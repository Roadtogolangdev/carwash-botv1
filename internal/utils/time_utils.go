package utils

import "fmt"

func FormatTime(hour int) string {
	return fmt.Sprintf("%d:00", hour)
}
