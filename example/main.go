package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

// CheckSignature is a function with a signature that fits in one line but is split across multiple lines.
func CheckSignature(
	a int,
	b int,
) {
	fmt.Println(a, b)
}

// CheckSignature2 is a function with a signature that is too long for one line but formatter keeps it
func CheckSignature2(
	param1 string,
	param2 string,
	param3 string,
	param4 string,
	param5 string,
	param6 string,
	param7 string,
) {
	fmt.Println(param1)
}
