package kotlin

import "strings"

// stripComments removes line (//) and block (/* */) comments from a line.
// It returns the stripped line and whether we are still inside a block comment.
func stripComments(line string, inBlock bool) (string, bool) {
	var out strings.Builder
	i := 0

	for i < len(line) {
		if inBlock {
			end := strings.Index(line[i:], "*/")
			if end == -1 {
				return out.String(), true
			}
			i += end + 2
			inBlock = false
			continue
		}

		if strings.HasPrefix(line[i:], "/*") {
			inBlock = true
			i += 2
			continue
		}
		if strings.HasPrefix(line[i:], "//") {
			break
		}

		out.WriteByte(line[i])
		i++
	}

	return out.String(), inBlock
}
