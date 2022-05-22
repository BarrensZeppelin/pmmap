package pmmap

func zeroBit(key, bit keyt) bool {
	return key & bit == 0
}

// branchingBit returns a number with a single bit set at the first position
// (least significant) where p0 and p1 differ.
func branchingBit(p0, p1 keyt) keyt {
	diff := p0 ^ p1
	return diff & -diff
}
