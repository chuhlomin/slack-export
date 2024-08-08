package utils

import (
	"strconv"
	"strings"
	"time"
)

func EpochToRFC1123(epochTime string) (string, error) {
	epochFloat, err := strconv.ParseFloat(strings.TrimSpace(epochTime), 64)
	if err != nil {
		return "", err
	}

	return time.Unix(int64(epochFloat), 0).UTC().Format(time.RFC1123), nil
}
