package uuidv8

import (
	"encoding/hex"
	"errors"
	"fmt"
)

// Helper function to encode timestamp into the UUID byte array.
func encodeTimestamp(uuid []byte, timestamp uint64, timestampBits int) error {
	switch timestampBits {
	case TimestampBits32:
		uuid[0], uuid[1], uuid[2], uuid[3] = byte(timestamp>>24), byte(timestamp>>16), byte(timestamp>>8), byte(timestamp)
		uuid[4], uuid[5] = 0, 0
	case TimestampBits48:
		uuid[0], uuid[1], uuid[2], uuid[3], uuid[4], uuid[5] = byte(timestamp>>40), byte(timestamp>>32), byte(timestamp>>24), byte(timestamp>>16), byte(timestamp>>8), byte(timestamp)
	case TimestampBits60:
		uuid[0], uuid[1], uuid[2], uuid[3], uuid[4], uuid[5], uuid[6] = byte(timestamp>>52), byte(timestamp>>44), byte(timestamp>>36), byte(timestamp>>28), byte(timestamp>>20), byte(timestamp>>12), byte(timestamp>>4)
	default:
		return fmt.Errorf("unsupported timestamp bit size: %d", timestampBits)
	}
	return nil
}

// Helper function to decode a timestamp from the UUID byte array.
func decodeTimestamp(uuidBytes []byte) uint64 {
	return uint64(uuidBytes[0])<<40 | uint64(uuidBytes[1])<<32 | uint64(uuidBytes[2])<<24 |
		uint64(uuidBytes[3])<<16 | uint64(uuidBytes[4])<<8 | uint64(uuidBytes[5])
}

// Helper function to parse and sanitize a UUID string.
func parseUUID(uuid string) ([]byte, error) {
	if len(uuid) == 32 {
		// Fast path for UUIDs without dashes
		return hex.DecodeString(uuid)
	} else if len(uuid) == 36 {
		// Validate dash positions
		if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
			return nil, errors.New("invalid UUID format")
		}
	} else {
		return nil, errors.New("invalid UUID length")
	}

	// Remove dashes while copying characters
	result := make([]byte, 32)
	j := 0
	for i := 0; i < len(uuid); i++ {
		if uuid[i] == '-' {
			continue
		}
		result[j] = uuid[i]
		j++
	}

	return hex.DecodeString(string(result))
}

// Helper function to check if a UUID is all zeros.
func isAllZeroUUID(uuidBytes []byte) bool {
	for _, b := range uuidBytes {
		if b != 0 {
			return false
		}
	}
	return true
}

// Helper function to format a UUID byte array as a string.
func formatUUID(uuid []byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
