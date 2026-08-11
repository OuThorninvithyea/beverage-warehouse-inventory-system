package catalog

func ValidateBarcode(value string) bool {
	if len(value) != 12 && len(value) != 13 {
		return false
	}

	sum := 0
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
		digit := int(value[i] - '0')
		if i == len(value)-1 {
			check := (10 - (sum % 10)) % 10
			return digit == check
		}
		if len(value) == 12 {
			if i%2 == 0 {
				sum += digit * 3
			} else {
				sum += digit
			}
			continue
		}
		if i%2 == 0 {
			sum += digit
		} else {
			sum += digit * 3
		}
	}
	return false
}
