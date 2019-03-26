package uint128

const digits = "0123456789"

// cutoff is the smallest n that n*10 will overlow an Int.
var cutoff = Int{1844674407370955161, 11068046444225730970}

// Itoa returns the string representation of x in base 10.
func Itoa(x Int) string {
	s := make([]byte, 0)
	for x.IsGreaterThan(Int{0, 10}) {
		quo, mod := div(x, Int{0, 10})

		s = append(s, digits[int(mod.Lo)])
		x = quo
	}
	s = append(s, digits[int(x.Lo)])

	// reverse the string
	for l, r := 0, len(s)-1; l < r; l, r = l+1, r-1 {
		s[l], s[r] = s[r], s[l]
	}

	return string(s)
}

// Atoi creates a new Int from s, interpreted in base 10.
func Atoi(s string) (Int, error) {
	if len(s) == 0 {
		return Zero, ErrInvalidString
	}

	x := Zero
	for _, c := range []byte(s) {
		if c < '0' || c > '9' {
			return Max, ErrInvalidString
		}
		digit := Int{0, uint64(c - '0')}

		if x.IsGreaterThan(cutoff) || x.IsEqualTo(cutoff) {
			return Max, ErrOverflow
		}
		x = x.Mul(Int{0, 10})

		x1 := x.Add(digit)
		if x1.IsLessThan(x) {
			return Max, ErrOverflow
		}
		x = x1
	}

	return x, nil
}
