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

// D! id=format range-start
func FormatUser(name string) string {
	return "User: " + name
}
// D! id=format range-end

// D! id=perms range-start
func CheckPermission(role string) bool {
	if role == "admin" {
		return true
	}
	return false
}
// D! id=perms range-end

func main() {
	fmt.Println(FormatUser("alice"))
}
