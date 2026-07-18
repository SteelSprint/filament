package main

import "fmt"

// D! id=validate range-start
func ValidateUser(name string) error {
	if name == "" {
		return fmt.Errorf("name must not be empty")
	}
	return nil
}
// D! id=validate range-end

func main() {
	fmt.Println(ValidateUser("alice"))
}
