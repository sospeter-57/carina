package main

import (
	"fmt"
	
	"carina/internal/config"
)
func main() {
	var _ = config.Load()
	fmt.Println("everything went well")
}
