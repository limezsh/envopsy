package mixed

import "os"

func F() {
	_ = os.Getenv("MIXED_GO")
}
