package slashing

func sameBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isSurrounding(sourceA, targetA, sourceB, targetB uint64) bool {
	return sourceA < sourceB && targetB < targetA
}

func isSurrounded(sourceA, targetA, sourceB, targetB uint64) bool {
	return sourceB < sourceA && targetA < targetB
}
