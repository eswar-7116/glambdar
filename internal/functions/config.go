package functions

func UpdateRateLimit(funcName string, limit int) error {
	md, err := LoadMetadata(funcName)
	if err != nil {
		return err
	}

	md.RateLimit = limit
	return SaveMetadata(md)
}
