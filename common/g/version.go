package g

import "fmt"
const (
	BinaryName = "Dynamic Application Configuration Management"
	Version = "v0.0.1"
)

func VersionInfo() string {
	return fmt.Sprintf("%s", Version)
}

func VersionDetailInfo() string {
	return fmt.Sprintf("%s %s", BinaryName, Version)
}
