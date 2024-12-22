package main

import (
	"net"
	"net/mail"
	"strings"
)

func ValidateEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	parts := strings.Split(addr.Address, "@")
	if len(parts) != 2 {
		return false
	}

	username := parts[0]
	domain := parts[1]

	mxRecords, err := net.LookupMX(domain)
	if err != nil || len(mxRecords) == 0 {
		return false
	}

	if len(username) == 0 || len(username) > 64 {
		return false
	}

	if strings.EqualFold(domain, "gmail.com") && len(username) < 6 {
		return false
	}

	return true
}
