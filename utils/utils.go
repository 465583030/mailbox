package utils

import (
	"encoding/hex"

	"github.com/google/uuid"
)

const (
	// SQLPATH is for SQL link path
	SQLPATH = "root:mailboxdbs@tcp(MARIADB:3306)/mailbox?parseTime=true"
)

// GenSeed is to gen seed
func GenSeed() [8]byte {
	var buf [8]byte
	u := uuid.Must(uuid.NewRandom())
	hex.Encode(buf[:], u[:4])
	return buf
}
