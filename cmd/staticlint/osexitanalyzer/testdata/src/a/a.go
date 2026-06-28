package a

import "os"

func main() {
	os.Exit(1)
}

func exit(code int) {
	os.Exit(code)
}
