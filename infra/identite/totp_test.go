package identite

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"time"
)

const (
	periodeOTP    = 30
	chiffresOTP   = 1_000_000
	formatCodeOTP = "%06d"
)

func valeurOTP(secret string, decalage int64) uint32 {
	var message bytes.Buffer
	_ = binary.Write(&message, binary.BigEndian, time.Now().Unix()/periodeOTP+decalage)
	empreinte := hmac.New(sha1.New, []byte(secret))
	empreinte.Write(message.Bytes())
	somme := empreinte.Sum(nil)
	position := somme[len(somme)-1] & 0x0f
	return (binary.BigEndian.Uint32(somme[position:position+4]) & 0x7fffffff) % chiffresOTP
}

func codeOTP(secret string, decalage int64) string {
	return fmt.Sprintf(formatCodeOTP, valeurOTP(secret, decalage))
}

func codeOTPFaux(secret string) string {
	return fmt.Sprintf(formatCodeOTP, (valeurOTP(secret, 0)+chiffresOTP/2)%chiffresOTP)
}
