package flipcamlib

func defaultString(value string, defaultVal string) string {
	if value == "" {
		return defaultVal
	}

	return value
}
