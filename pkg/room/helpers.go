package room

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"
)

func createID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b[:24]); err != nil {
		return "", fmt.Errorf("creating ID: %w", err)
	}

	// 8 bytes used for the timestamp i.e. 32 - 8 = 24
	binary.BigEndian.PutUint64(b[24:], uint64(time.Now().UnixMilli()))
	return hex.EncodeToString(b), nil
}
