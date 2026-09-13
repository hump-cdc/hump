package funcs

import "math"

// tmplAdd returns the sum of all integer arguments.
func tmplAdd(nums ...any) int64 {
	var sum int64
	for _, n := range nums {
		sum += tmplInt64(n)
	}
	return sum
}

// tmplAdd1 returns v+1 as int64.
func tmplAdd1(v any) int64 { return tmplInt64(v) + 1 }

// tmplSub returns a-b.
func tmplSub(a, b any) int64 { return tmplInt64(a) - tmplInt64(b) }

// tmplMul returns the product of all integer arguments.
func tmplMul(nums ...any) int64 {
	if len(nums) == 0 {
		return 0
	}
	prod := int64(1)
	for _, n := range nums {
		prod *= tmplInt64(n)
	}
	return prod
}

// tmplDiv returns a/b (integer division). b=0 yields 0.
func tmplDiv(a, b any) int64 {
	bv := tmplInt64(b)
	if bv == 0 {
		return 0
	}
	return tmplInt64(a) / bv
}

// tmplMod returns a%b. b=0 yields 0.
func tmplMod(a, b any) int64 {
	bv := tmplInt64(b)
	if bv == 0 {
		return 0
	}
	return tmplInt64(a) % bv
}

// tmplMax returns the largest of nums.
func tmplMax(nums ...any) int64 {
	if len(nums) == 0 {
		return 0
	}
	m := tmplInt64(nums[0])
	for _, n := range nums[1:] {
		v := tmplInt64(n)
		if v > m {
			m = v
		}
	}
	return m
}

// tmplMin returns the smallest of nums.
func tmplMin(nums ...any) int64 {
	if len(nums) == 0 {
		return 0
	}
	m := tmplInt64(nums[0])
	for _, n := range nums[1:] {
		v := tmplInt64(n)
		if v < m {
			m = v
		}
	}
	return m
}

// tmplAddF returns the sum of all float arguments.
func tmplAddF(nums ...any) float64 {
	var sum float64
	for _, n := range nums {
		sum += tmplFloat64(n)
	}
	return sum
}

// tmplAdd1F returns v+1 as float64.
func tmplAdd1F(v any) float64 { return tmplFloat64(v) + 1 }

// tmplSubF returns a-b as float64.
func tmplSubF(a, b any) float64 { return tmplFloat64(a) - tmplFloat64(b) }

// tmplMulF returns the product of all float arguments.
func tmplMulF(nums ...any) float64 {
	if len(nums) == 0 {
		return 0
	}
	prod := 1.0
	for _, n := range nums {
		prod *= tmplFloat64(n)
	}
	return prod
}

// tmplDivF returns a/b as float64.
func tmplDivF(a, b any) float64 {
	bv := tmplFloat64(b)
	if bv == 0 {
		return 0
	}
	return tmplFloat64(a) / bv
}

// tmplMaxF returns the largest of nums (float64).
func tmplMaxF(nums ...any) float64 {
	if len(nums) == 0 {
		return 0
	}
	m := tmplFloat64(nums[0])
	for _, n := range nums[1:] {
		v := tmplFloat64(n)
		if v > m {
			m = v
		}
	}
	return m
}

// tmplMinF returns the smallest of nums (float64).
func tmplMinF(nums ...any) float64 {
	if len(nums) == 0 {
		return 0
	}
	m := tmplFloat64(nums[0])
	for _, n := range nums[1:] {
		v := tmplFloat64(n)
		if v < m {
			m = v
		}
	}
	return m
}

// tmplFloor returns the largest integer ≤ v as float64.
func tmplFloor(v any) float64 { return math.Floor(tmplFloat64(v)) }

// tmplCeil returns the smallest integer ≥ v as float64.
func tmplCeil(v any) float64 { return math.Ceil(tmplFloat64(v)) }

// tmplRound rounds v to the given decimal precision. precision=0 rounds to
// the nearest integer.
func tmplRound(v any, precision int) float64 {
	f := tmplFloat64(v)
	shift := math.Pow(10, float64(precision))
	return math.Round(f*shift) / shift
}
