package strings

func StringPtr(val string) *string {
	if val == "" {
		return nil
	}
	return &val
}
