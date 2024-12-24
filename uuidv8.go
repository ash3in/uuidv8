// Package uuidv8 provides utilities for generating, parsing, validating,
// and converting UUIDv8 (based on the UUIDv8 specification) and related components.
package uuidv8

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Constants for the variant and version of UUIDs based on the RFC4122 specification.
const (
	variantRFC4122 = 0b10 // Variant bits for RFC4122
	versionV8      = 0x8  // Version bits for UUIDv8
)

// Supported timestamp bit sizes for UUIDv8.
const (
	TimestampBits32 = 32 // Use 32-bit timestamp
	TimestampBits48 = 48 // Use 48-bit timestamp
	TimestampBits60 = 60 // Use 60-bit timestamp
)

// UUIDv8 represents a parsed UUIDv8 object.
//
// Fields:
// - Timestamp: Encoded timestamp value (up to 60 bits).
// - ClockSeq: Clock sequence value (up to 12 bits).
// - Node: Node value, typically a 6-byte unique identifier.
type UUIDv8 struct {
	Timestamp uint64 // The timestamp component of the UUID.
	ClockSeq  uint16 // The clock sequence component of the UUID.
	Node      []byte // The node component of the UUID (typically 6 bytes).
}

// New generates a UUIDv8 with default parameters.
//
// Default behavior:
// - Timestamp: Current time in nanoseconds.
// - ClockSeq: Random 12-bit value.
// - Node: Random 6-byte node identifier.
//
// Returns:
// - A string representation of the generated UUIDv8.
// - An error if any component generation fails.
func New() (string, error) {
	// Current timestamp
	timestamp := uint64(time.Now().UnixNano())

	// Random clock sequence
	clockSeq := make([]byte, 2)
	if _, err := rand.Read(clockSeq); err != nil {
		return "", fmt.Errorf("failed to generate random clock sequence: %w", err)
	}
	clockSeqValue := binary.BigEndian.Uint16(clockSeq) & 0x0FFF // Mask to 12 bits

	// Random node
	node := make([]byte, 6)
	if _, err := rand.Read(node); err != nil {
		return "", fmt.Errorf("failed to generate random node: %w", err)
	}

	// Generate UUIDv8
	return NewWithParams(timestamp, clockSeqValue, node, TimestampBits48)
}

// NewWithParams generates a new UUIDv8 based on the provided timestamp, clock sequence, and node.
//
// Parameters:
// - timestamp: A 32-, 48-, or 60-bit timestamp value (depending on `timestampBits`).
// - clockSeq: A 12-bit clock sequence value for sequencing UUIDs generated within the same timestamp.
// - node: A 6-byte slice representing a unique identifier (e.g., MAC address or random bytes).
// - timestampBits: The number of bits in the timestamp (32, 48, or 60).
//
// Returns:
// - A string representation of the generated UUIDv8.
// - An error if the input parameters are invalid (e.g., incorrect node length or unsupported timestamp size).
func NewWithParams(timestamp uint64, clockSeq uint16, node []byte, timestampBits int) (string, error) {
	if len(node) != 6 {
		return "", fmt.Errorf("node must be 6 bytes, got %d bytes", len(node))
	}

	uuid := make([]byte, 16)

	// Set timestamp
	if err := encodeTimestamp(uuid, timestamp, timestampBits); err != nil {
		return "", err
	}

	// Set version and clock sequence
	uuid[6] = (byte(versionV8) << 4) | byte(clockSeq>>8)
	uuid[7] = byte(clockSeq)

	// Set variant
	uuid[7] = (uuid[7] & 0x3F) | (variantRFC4122 << 6)

	// Copy node
	copy(uuid[8:], node)

	return formatUUID(uuid), nil
}

// FromString parses a UUIDv8 string into its components.
//
// Parameters:
// - uuid: A string representation of a UUIDv8.
//
// Returns:
// - A pointer to a UUIDv8 struct containing the parsed components (timestamp, clockSeq, node).
// - An error if the UUID is invalid or cannot be parsed.
func FromString(uuid string) (*UUIDv8, error) {
	uuidBytes, err := parseUUID(uuid)
	if err != nil {
		return nil, fmt.Errorf("failed to parse UUID: %w", err)
	}

	// Decode timestamp (48 bits default, adjust for compatibility)
	timestamp := uint64(uuidBytes[0])<<40 |
		uint64(uuidBytes[1])<<32 |
		uint64(uuidBytes[2])<<24 |
		uint64(uuidBytes[3])<<16 |
		uint64(uuidBytes[4])<<8 |
		uint64(uuidBytes[5])

	// Decode clock sequence (12 bits)
	clockSeq := uint16(uuidBytes[6]&0x0F)<<8 | uint16(uuidBytes[7])

	// Decode node (last 6 bytes)
	node := uuidBytes[8:14]

	return &UUIDv8{
		Timestamp: timestamp,
		ClockSeq:  clockSeq,
		Node:      node,
	}, nil
}

// FromStringOrNil parses a UUIDv8 string into its components, returning nil if invalid or all zero.
//
// Parameters:
// - uuid: A string representation of a UUIDv8.
//
// Returns:
// - A pointer to a UUIDv8 struct if the UUID is valid.
// - Nil if the UUID is invalid or represents an all-zero UUID.
func FromStringOrNil(uuid string) *UUIDv8 {
	uuidBytes, err := parseUUID(uuid)
	if err != nil || isAllZeroUUID(uuidBytes) {
		return nil
	}

	timestamp := decodeTimestamp(uuidBytes[:6])
	clockSeq := uint16(uuidBytes[6]&0x0F)<<8 | uint16(uuidBytes[7])
	node := uuidBytes[8:14]

	return &UUIDv8{
		Timestamp: timestamp,
		ClockSeq:  clockSeq,
		Node:      node,
	}
}

// IsValidUUIDv8 validates if a given string is a valid UUIDv8.
//
// Parameters:
// - uuid: A string representation of a UUID.
//
// Returns:
// - A boolean indicating whether the UUID is valid.
//   - `true` if the UUID has the correct version and variant bits and is well-formed.
//   - `false` if the UUID is invalid or all zero.
func IsValidUUIDv8(uuid string) bool {
	uuidBytes, err := parseUUID(uuid)
	if err != nil || isAllZeroUUID(uuidBytes) {
		return false
	}

	version := uuidBytes[6] >> 4
	variant := (uuidBytes[7] >> 6) & 0x03

	return version == versionV8 && variant == variantRFC4122
}

// ToString converts a UUIDv8 struct into its string representation.
//
// Parameters:
// - uuidv8: A pointer to a UUIDv8 struct containing the components (timestamp, clockSeq, node).
//
// Returns:
// - A string representation of the UUIDv8.
func ToString(uuidv8 *UUIDv8) string {
	uuid := make([]byte, 16)

	// Encode timestamp
	err := encodeTimestamp(uuid, uuidv8.Timestamp, TimestampBits48)
	if err != nil {
		return ""
	}

	// Set clock sequence and version
	uuid[6] = (byte(versionV8) << 4) | byte(uuidv8.ClockSeq>>8)
	uuid[7] = byte(uuidv8.ClockSeq)

	// Set variant
	uuid[7] = (uuid[7] & 0x3F) | (variantRFC4122 << 6)

	// Copy node
	copy(uuid[8:], uuidv8.Node)

	return formatUUID(uuid)
}

// MarshalJSON serializes a UUIDv8 object into its JSON representation.
//
// Returns:
// - A JSON-encoded byte slice of the UUID string.
// - An error if the serialization fails.
func (u *UUIDv8) MarshalJSON() ([]byte, error) {
	// Validate the UUIDv8 object before conversion
	if u == nil || len(u.Node) != 6 || u.Timestamp == 0 || u.ClockSeq > 0x0FFF {
		return nil, fmt.Errorf("object is not a valid UUIDv8")
	}

	// Convert to string and validate
	uuidStr := ToString(u)
	if !IsValidUUIDv8(uuidStr) {
		return nil, fmt.Errorf("string representation is not a valid UUIDv8")
	}

	return json.Marshal(uuidStr)
}

// UnmarshalJSON deserializes a JSON-encoded UUIDv8 string into a UUIDv8 object.
//
// Parameters:
// - data: A JSON-encoded byte slice containing the UUID string.
//
// Returns:
// - An error if the deserialization fails or if the UUID string is invalid.
func (u *UUIDv8) UnmarshalJSON(data []byte) error {
	var uuidStr string
	if err := json.Unmarshal(data, &uuidStr); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Ensure the UUID string is valid and represents a UUIDv8
	if !IsValidUUIDv8(uuidStr) {
		return fmt.Errorf("input is not a valid UUIDv8: %s", uuidStr)
	}

	parsed, err := FromString(uuidStr)
	if err != nil {
		return fmt.Errorf("failed to parse UUID string: %w", err)
	}

	*u = *parsed
	return nil
}

// Value implements the driver.Value interface for database writes.
func (u *UUIDv8) Value() (driver.Value, error) {
	if u == nil || len(u.Node) != 6 {
		return nil, nil
	}
	return ToString(u), nil
}

// Scan implements the interface for database reads.
func (u *UUIDv8) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		parsed, err := FromString(v)
		if err != nil {
			return err
		}
		*u = *parsed
	case []byte:
		parsed, err := FromString(string(v))
		if err != nil {
			return err
		}
		*u = *parsed
	default:
		return errors.New("unsupported type for UUIDv8")
	}
	return nil
}
