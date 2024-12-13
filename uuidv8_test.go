package uuidv8_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/ash3in/uuidv8"
)

func TestNewUUIDv8(t *testing.T) {
	node := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	timestamp := uint64(1633024800000000000) // Fixed timestamp for deterministic tests
	clockSeq := uint16(0)

	tests := []struct {
		timestampBits int
		expectedErr   bool
		description   string
	}{
		{uuidv8.TimestampBits32, false, "32-bit timestamp"},
		{uuidv8.TimestampBits48, false, "48-bit timestamp"},
		{uuidv8.TimestampBits60, false, "60-bit timestamp"},
		{100, true, "Invalid timestamp bit size"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			uuid, err := uuidv8.NewUUIDv8(timestamp, clockSeq, node, test.timestampBits)
			if (err != nil) != test.expectedErr {
				t.Errorf("Expected error: %v, got: %v", test.expectedErr, err)
			}

			if err == nil && uuid == "" {
				t.Error("Generated UUID is empty")
			}
		})
	}
}

func TestNewUUIDv8_NodeValidation(t *testing.T) {
	invalidNodes := [][]byte{
		nil,          // Nil node
		{},           // Empty node
		{0x01, 0x02}, // Less than 6 bytes
		{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}, // More than 6 bytes
	}

	for _, node := range invalidNodes {
		t.Run("Invalid node length", func(t *testing.T) {
			_, err := uuidv8.NewUUIDv8(1633024800, 0, node, uuidv8.TimestampBits48)
			if err == nil {
				t.Errorf("Expected error for invalid node: %v", node)
			}
		})
	}
}

func TestIsValidUUIDv8(t *testing.T) {
	tests := []struct {
		uuid        string
		shouldPass  bool
		description string
	}{
		{"00000000-0000-0000-0000-000000000000", false, "All-zero UUIDv8"},
		{"9a3d4049-0e2c-7080-0102-030405060000", false, "Incorrect version"},
		{"9a3d4049-0e2c-9080-0102-030405060000", false, "Incorrect variant"},
		{"invalid-uuid", false, "Invalid UUID format"},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			valid := uuidv8.IsValidUUIDv8(test.uuid)
			if valid != test.shouldPass {
				t.Errorf("Validation mismatch for UUIDv8 %s: expected %v, got %v", test.uuid, test.shouldPass, valid)
			}
		})
	}
}

func TestFromString(t *testing.T) {
	node := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	timestamp := uint64(1633024800000000000) // Fixed timestamp
	clockSeq := uint16(0)

	uuid, err := uuidv8.NewUUIDv8(timestamp, clockSeq, node, uuidv8.TimestampBits48)
	if err != nil {
		t.Fatalf("NewUUIDv8 failed: %v", err)
	}

	parsed, err := uuidv8.FromString(uuid)
	if err != nil {
		t.Errorf("FromString failed: %v", err)
	}

	if parsed == nil {
		t.Error("FromString returned nil for a valid UUID")
	}

	// Validate parsed fields (adjust expectations based on bit shifts in encoding)
	expectedTimestamp := timestamp & ((1 << 48) - 1) // Mask to match the 48 bits encoded
	if parsed.Timestamp != expectedTimestamp {
		t.Errorf("Parsed timestamp mismatch: expected %d, got %d", expectedTimestamp, parsed.Timestamp)
	}
	if len(parsed.Node) != 6 {
		t.Errorf("Parsed node length mismatch: expected 6, got %d", len(parsed.Node))
	}

	_, err = uuidv8.FromString("invalid-uuid")
	if err == nil {
		t.Error("FromString failed: did not return an error for an invalid UUID")
	}
}

func TestFromStringOrNil(t *testing.T) {
	// Valid UUIDv8 string
	validUUID := "9a3d4049-0e2c-8080-0102-030405060000"
	parsedUUID := uuidv8.FromStringOrNil(validUUID)

	if parsedUUID == nil {
		t.Errorf("FromStringOrNil failed: returned nil for a valid UUID %s", validUUID)
	} else {
		// Validate parsed fields
		if parsedUUID.Timestamp == 0 {
			t.Errorf("FromStringOrNil failed: Timestamp is not parsed correctly for UUID %s", validUUID)
		}
		if len(parsedUUID.Node) != 6 {
			t.Errorf("FromStringOrNil failed: Node length is incorrect for UUID %s", validUUID)
		}
	}

	// Invalid UUIDv8 string
	invalidUUID := "invalid-uuid"
	parsedInvalid := uuidv8.FromStringOrNil(invalidUUID)

	if parsedInvalid != nil {
		t.Errorf("FromStringOrNil failed: returned a non-nil object for an invalid UUID %s", invalidUUID)
	}

	// All-zero UUID
	allZeroUUID := "00000000-0000-0000-0000-000000000000"
	parsedZero := uuidv8.FromStringOrNil(allZeroUUID)

	if parsedZero != nil {
		t.Errorf("FromStringOrNil failed: returned a non-nil object for an all-zero UUID %s", allZeroUUID)
	}
}

func TestConcurrencySafety(t *testing.T) {
	node := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	const concurrencyLevel = 100

	var wg sync.WaitGroup
	uuidSet := sync.Map{}

	start := time.Now()

	for i := 0; i < concurrencyLevel; i++ {
		wg.Add(1)

		go func(clockSeq uint16, index int) {
			defer wg.Done()

			timestamp := uint64(time.Now().UnixNano()) + uint64(index)

			uuid, err := uuidv8.NewUUIDv8(timestamp, clockSeq, node, uuidv8.TimestampBits48)
			if err != nil {
				t.Errorf("Failed to generate UUIDv8 in concurrent environment: %v", err)
			}

			uuidSet.Store(uuid, true)
		}(uint16(i), i)
	}

	wg.Wait()

	// Measure time taken
	elapsed := time.Since(start)
	t.Logf("Concurrency test completed in %s", elapsed)

	count := 0
	uuidSet.Range(func(_, _ interface{}) bool {
		count++
		return true
	})

	if count != concurrencyLevel {
		t.Errorf("Expected %d unique UUIDs, but got %d", concurrencyLevel, count)
	}
}

func TestEdgeCases(t *testing.T) {
	node := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}

	t.Run("Minimum timestamp and clock sequence", func(t *testing.T) {
		uuid, err := uuidv8.NewUUIDv8(0, 0, node, uuidv8.TimestampBits48)
		if err != nil || uuid == "" {
			t.Error("Failed to generate UUID with minimal timestamp and clock sequence")
		}
	})

	t.Run("Maximum timestamp and clock sequence", func(t *testing.T) {
		maxTimestamp := uint64(1<<48 - 1)
		maxClockSeq := uint16(1<<12 - 1)
		uuid, err := uuidv8.NewUUIDv8(maxTimestamp, maxClockSeq, node, uuidv8.TimestampBits48)
		if err != nil || uuid == "" {
			t.Error("Failed to generate UUID with maximum timestamp and clock sequence")
		}

		if !uuidv8.IsValidUUIDv8(uuid) {
			t.Errorf("IsValidUUIDv8 failed: UUID %s is invalid", uuid)
		}
	})
}

func TestMarshalJSON(t *testing.T) {
	node := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	timestamp := uint64(1633024800000000000) // Fixed timestamp
	clockSeq := uint16(0)

	// Generate a valid UUIDv8
	uuidStr, err := uuidv8.NewUUIDv8(timestamp, clockSeq, node, uuidv8.TimestampBits48)
	if err != nil {
		t.Fatalf("Failed to generate UUIDv8: %v", err)
	}

	parsedUUID, err := uuidv8.FromString(uuidStr)
	if err != nil {
		t.Fatalf("Failed to parse UUIDv8: %v", err)
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(parsedUUID)
	if err != nil {
		t.Errorf("Failed to marshal UUIDv8 to JSON: %v", err)
	}

	// Validate JSON output
	expectedJSON := `"` + uuidStr + `"`
	if string(jsonData) != expectedJSON {
		t.Errorf("JSON output mismatch: expected %s, got %s", expectedJSON, string(jsonData))
	}
}

func TestMarshalInvalidUUID(t *testing.T) {
	// Invalid UUID with incorrect clock sequence and node length
	invalidUUIDs := []*uuidv8.UUIDv8{
		{Timestamp: 0, ClockSeq: 100, Node: []byte{0x01, 0x02, 0x03}},                        // Invalid node length
		{Timestamp: 123, ClockSeq: 0x1000, Node: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}}, // Invalid clock sequence
		{Timestamp: 0, ClockSeq: 0, Node: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}},        // Invalid timestamp
	}

	for _, invalidUUID := range invalidUUIDs {
		_, err := json.Marshal(invalidUUID)
		if err == nil {
			t.Errorf("Expected error when marshalling invalid UUIDv8: %+v", invalidUUID)
		}
	}
}

func TestMarshalValidUUID(t *testing.T) {
	// Valid UUID
	validUUID := &uuidv8.UUIDv8{
		Timestamp: 123456789,
		ClockSeq:  0x0800,
		Node:      []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
	}

	data, err := json.Marshal(validUUID)
	if err != nil {
		t.Errorf("Failed to marshal valid UUIDv8: %v", err)
	}

	// Dynamically calculate the expected string
	expected := uuidv8.ToString(validUUID)
	if string(data) != `"`+expected+`"` {
		t.Errorf("Expected JSON %s, got %s", `"`+expected+`"`, string(data))
	}
}

func TestUnmarshalJSON(t *testing.T) {
	node := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	timestamp := uint64(1633024800000000000) // Fixed timestamp
	clockSeq := uint16(0)

	// Generate a valid UUIDv8
	uuidStr, err := uuidv8.NewUUIDv8(timestamp, clockSeq, node, uuidv8.TimestampBits48)
	if err != nil {
		t.Fatalf("Failed to generate UUIDv8: %v", err)
	}

	// JSON string representing the UUID
	jsonData := `"` + uuidStr + `"`

	// Unmarshal JSON
	var parsedUUID uuidv8.UUIDv8
	err = json.Unmarshal([]byte(jsonData), &parsedUUID)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON to UUIDv8: %v", err)
	}

	// Validate parsed fields
	expectedUUID, _ := uuidv8.FromString(uuidStr)
	if parsedUUID.Timestamp != expectedUUID.Timestamp {
		t.Errorf("Timestamp mismatch: expected %d, got %d", expectedUUID.Timestamp, parsedUUID.Timestamp)
	}
	if parsedUUID.ClockSeq != expectedUUID.ClockSeq {
		t.Errorf("ClockSeq mismatch: expected %d, got %d", expectedUUID.ClockSeq, parsedUUID.ClockSeq)
	}
	if len(parsedUUID.Node) != len(expectedUUID.Node) {
		t.Errorf("Node length mismatch: expected %d, got %d", len(expectedUUID.Node), len(parsedUUID.Node))
	}
	for i := range parsedUUID.Node {
		if parsedUUID.Node[i] != expectedUUID.Node[i] {
			t.Errorf("Node byte mismatch at index %d: expected %x, got %x", i, expectedUUID.Node[i], parsedUUID.Node[i])
		}
	}
}

func TestUnmarshalInvalidJSON(t *testing.T) {
	// Invalid JSON input
	invalidJSONs := []string{
		`"invalid-uuid"`,                         // Invalid UUID string
		`"0193bde4-a9fa-77eb-a304-6cf8530ece78"`, // A UUIDv7
		`"12345"`,                                // Incorrect length
		`"00000000-0000-0000-0000-000000000000"`, // All-zero UUID
	}

	for _, jsonData := range invalidJSONs {
		var parsedUUID uuidv8.UUIDv8
		err := json.Unmarshal([]byte(jsonData), &parsedUUID)
		if err == nil {
			t.Errorf("Expected error for invalid JSON input: %s", jsonData)
		}
	}
}