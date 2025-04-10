package crypt

func SplitSignature(b []byte) ([]byte, []byte) {
	sigStart := len(b) - 64
	return b[:sigStart], b[sigStart:]
}
