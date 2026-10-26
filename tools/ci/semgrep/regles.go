package essai

import (
	"crypto/tls"

	// ruleid: go-regex-retour-arriere-interdite
	"github.com/dlclark/regexp2"
	// ok: go-regex-retour-arriere-interdite
	"regexp"
)

// ruleid: go-cgo-interdit
import "C"

func configurations() {
	_ = regexp.MustCompile("a+")
	_ = regexp2.MustCompile("a+", 0)
	// ruleid: go-verification-contournee
	_ = &tls.Config{InsecureSkipVerify: true}
	// ok: go-verification-contournee
	_ = &tls.Config{InsecureSkipVerify: false}
	// ruleid: go-verification-contournee
	_ = verificateur{SkipExpiryCheck: true}
}
