package slip39

func generators() [10]int {
	return [10]int{0xE0E040, 0x1C1C080, 0x3838100, 0x7070200, 0xE0E0009, 0x1C0C2412, 0x38086C24, 0x3090FC48, 0x21B1F890, 0x3F3F120}
}

func customization(extendable bool) []int {
	text := "shamir"
	if extendable {
		text = "shamir_extendable"
	}
	out := make([]int, len(text))
	for i := range text {
		out[i] = int(text[i])
	}
	return out
}

func polymod(values []int) int {
	gen := generators()
	chk := 1
	for _, value := range values {
		top := chk >> 20
		chk = (chk&0xFFFFF)<<radixBits ^ value
		for i, g := range gen {
			if top>>i&1 == 1 {
				chk ^= g
			}
		}
	}
	return chk
}

func createChecksum(custom, data []int) []int {
	values := append(append(append([]int{}, custom...), data...), make([]int, checksumWords)...)
	sum := polymod(values) ^ 1
	out := make([]int, checksumWords)
	for i := range out {
		out[i] = sum >> (radixBits * (checksumWords - 1 - i)) & (1<<radixBits - 1)
	}
	return out
}

func verifyChecksum(custom, data []int) bool {
	return polymod(append(append([]int{}, custom...), data...)) == 1
}
