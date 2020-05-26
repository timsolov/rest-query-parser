package rqp

import "strings"

func cleanSliceString(list []string) []string {
	var clean []string
	for _, v := range list {
		v = strings.Trim(v, " \t")
		if len(v) > 0 {
			clean = append(clean, v)
		}
	}
	return clean
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
