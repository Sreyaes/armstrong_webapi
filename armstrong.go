package user

func isArmstrongNumber(n int) bool {
	if n <= 0 {
		return false
	}

	original := n
	sum := 0
	digits := 0
	temp := n

	// Count digits
	for temp > 0 {
		digits++
		temp /= 10
	}

	// Calculate sum of powers
	temp = n
	for temp > 0 {
		digit := temp % 10
		power := 1
		for i := 0; i < digits; i++ {
			power *= digit
		}
		sum += power
		temp /= 10
	}

	return sum == original
}
