// This is an easy way to turn on/off depending on whether or not it is a
// devel or release build.

// +build !release
package base

func IsDevel() bool { return true }
