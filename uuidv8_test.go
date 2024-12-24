package uuidv8_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/ash3in/uuidv8"
)

func TestNew_DefaultBehavior(t *testing.T) {
	t.Run("Generate UUIDv8 with default settings", func(t *testing.T) {
		uuid, err := uuidv8.New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		// Check if the UUID is valid
		if !uuidv8.IsValidUUIDv8(uuid) {
			t.Errorf("New() generated an invalid UUID: %s", uuid)
		}
	})
}

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
			uuid, err := uuidv8.NewWithParams(timestamp, clockSeq, node, test.timestampBits)
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
			_, err := uuidv8.NewWithParams(1633024800, 0, node, uuidv8.TimestampBits48)
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

	uuid, err := uuidv8.NewWithParams(timestamp, clockSeq, node, uuidv8.TimestampBits48)
	if err != nil {
		t.Fatalf("NewWithParams failed: %v", err)
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

			uuid, err := uuidv8.NewWithParams(timestamp, clockSeq, node, uuidv8.TimestampBits48)
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
		uuid, err := uuidv8.NewWithParams(0, 0, node, uuidv8.TimestampBits48)
		if err != nil || uuid == "" {
			t.Error("Failed to generate UUID with minimal timestamp and clock sequence")
		}
	})

	t.Run("Maximum timestamp and clock sequence", func(t *testing.T) {
		maxTimestamp := uint64(1<<48 - 1)
		maxClockSeq := uint16(1<<12 - 1)
		uuid, err := uuidv8.NewWithParams(maxTimestamp, maxClockSeq, node, uuidv8.TimestampBits48)
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
	uuidStr, err := uuidv8.NewWithParams(timestamp, clockSeq, node, uuidv8.TimestampBits48)
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
	uuidStr, err := uuidv8.NewWithParams(timestamp, clockSeq, node, uuidv8.TimestampBits48)
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

func TestNew_Uniqueness(t *testing.T) {
	const numUUIDs = 1000
	uuidSet := make(map[string]struct{})

	for i := 0; i < numUUIDs; i++ {
		uuid, err := uuidv8.New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		if _, exists := uuidSet[uuid]; exists {
			t.Errorf("Duplicate UUID generated: %s", uuid)
		}
		uuidSet[uuid] = struct{}{}
	}

	if len(uuidSet) != numUUIDs {
		t.Errorf("Expected %d unique UUIDs, but got %d", numUUIDs, len(uuidSet))
	}
}

func TestNew_ConcurrencySafety(t *testing.T) {
	const concurrencyLevel = 100
	var wg sync.WaitGroup
	uuidSet := sync.Map{}

	for i := 0; i < concurrencyLevel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			uuid, err := uuidv8.New()
			if err != nil {
				t.Errorf("New() failed in concurrent environment: %v", err)
			}
			uuidSet.Store(uuid, true)
		}()
	}

	wg.Wait()

	// Verify uniqueness
	count := 0
	uuidSet.Range(func(_, _ interface{}) bool {
		count++
		return true
	})

	if count != concurrencyLevel {
		t.Errorf("Expected %d unique UUIDs, but got %d", concurrencyLevel, count)
	}
}

func TestNew_IntegrationWithParsing(t *testing.T) {
	uuid, err := uuidv8.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	parsed, err := uuidv8.FromString(uuid)
	if err != nil {
		t.Errorf("FromString failed to parse UUID generated by New(): %v", err)
	}

	if parsed == nil {
		t.Error("Parsed UUID is nil")
	}
}

func TestNew_EdgeCases(t *testing.T) {
	t.Run("Minimal possible timestamp and clock sequence", func(t *testing.T) {
		uuid, err := uuidv8.New()
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}

		parsed, _ := uuidv8.FromString(uuid)
		if parsed.Timestamp == 0 || parsed.ClockSeq == 0 {
			t.Errorf("New() generated UUID with invalid minimal values: %s", uuid)
		}
	})
}

func TestNew_JSONSerializationIntegration(t *testing.T) {
	uuid, err := uuidv8.New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(uuid)
	if err != nil {
		t.Errorf("Failed to marshal UUID to JSON: %v", err)
	}

	// Deserialize from JSON
	var parsedUUID uuidv8.UUIDv8
	err = json.Unmarshal(jsonData, &parsedUUID)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON to UUIDv8: %v", err)
	}

	// Ensure the deserialized UUID matches the original
	if uuidv8.ToString(&parsedUUID) != uuid {
		t.Errorf("Mismatch between original and deserialized UUID: original %s, deserialized %s", uuid, uuidv8.ToString(&parsedUUID))
	}
}

func TestEncodeTimestamp_InvalidTimestampBits(t *testing.T) {
	invalidBits := []int{0, 16, 64} // Unsupported timestamp bit sizes
	for _, bits := range invalidBits {
		t.Run("Invalid timestamp bits", func(t *testing.T) {
			node := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
			_, err := uuidv8.NewWithParams(1633024800, 0, node, bits)
			if err == nil {
				t.Errorf("Expected error for invalid timestamp bits: %d", bits)
			}
		})
	}
}

func TestEncodeTimestamp_BoundaryValues(t *testing.T) {
	boundaryTimestamps := []struct {
		timestamp uint64
		bits      int
	}{
		{timestamp: 0, bits: uuidv8.TimestampBits32}, // Minimal 32-bit
		{timestamp: (1 << 32) - 1, bits: uuidv8.TimestampBits32},
		{timestamp: 0, bits: uuidv8.TimestampBits48}, // Minimal 48-bit
		{timestamp: (1 << 48) - 1, bits: uuidv8.TimestampBits48},
		{timestamp: 0, bits: uuidv8.TimestampBits60}, // Minimal 60-bit
		{timestamp: (1 << 60) - 1, bits: uuidv8.TimestampBits60},
	}
	for _, test := range boundaryTimestamps {
		t.Run("Boundary timestamp", func(t *testing.T) {
			node := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
			uuid, err := uuidv8.NewWithParams(test.timestamp, 0, node, test.bits)
			if err != nil {
				t.Errorf("Failed to generate UUID with timestamp %d and bits %d: %v", test.timestamp, test.bits, err)
			}
			if !uuidv8.IsValidUUIDv8(uuid) {
				t.Errorf("Generated UUID is invalid: %s", uuid)
			}
		})
	}
}

func TestParseUUID_InvalidFormats(t *testing.T) {
	invalidUUIDs := []string{
		"1234",                 // Too short
		"gibberish-not-a-uuid", // Invalid characters
	}
	for _, uuid := range invalidUUIDs {
		t.Run("Invalid format", func(t *testing.T) {
			parsed := uuidv8.FromStringOrNil(uuid)
			if parsed != nil {
				t.Errorf("Expected nil for invalid UUID format: %s", uuid)
			}
		})
	}
}

func TestFromString_ErrorScenarios(t *testing.T) {
	t.Run("Invalid UUID string", func(t *testing.T) {
		_, err := uuidv8.FromString("not-a-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID string")
		}
	})
}

func TestToString_EmptyUUID(t *testing.T) {
	emptyUUID := &uuidv8.UUIDv8{} // Uninitialized UUIDv8
	if result := uuidv8.ToString(emptyUUID); result != "00000000-0000-8080-0000-000000000000" {
		t.Errorf("Expected empty string for uninitialized UUID, got %s", result)
	}
}

func TestMarshalJSON_ErrorCases(t *testing.T) {
	invalidUUID := &uuidv8.UUIDv8{
		Timestamp: 0,
		ClockSeq:  0x1000,                   // Invalid clock sequence (exceeds 12 bits)
		Node:      []byte{0x01, 0x02, 0x03}, // Invalid node length
	}
	_, err := invalidUUID.MarshalJSON()
	if err == nil {
		t.Errorf("Expected error for invalid UUID in MarshalJSON")
	}
}

func TestUnmarshalJSON_ErrorCases(t *testing.T) {
	invalidJSON := []byte(`"not-a-uuid"`)
	var uuid uuidv8.UUIDv8
	if err := uuid.UnmarshalJSON(invalidJSON); err == nil {
		t.Error("Expected error for invalid JSON input")
	}
}

func TestNewWithParams_MaxValues(t *testing.T) {
	node := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	timestamp := uint64(1<<60 - 1)
	clockSeq := uint16(1<<12 - 1)

	uuid, err := uuidv8.NewWithParams(timestamp, clockSeq, node, uuidv8.TimestampBits60)
	if err != nil {
		t.Fatalf("Failed to generate UUID with max values: %v", err)
	}

	if !uuidv8.IsValidUUIDv8(uuid) {
		t.Errorf("Generated UUID with max values is invalid: %s", uuid)
	}
}

func TestUUIDv8_Value(t *testing.T) {
	// Mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	uuid := &uuidv8.UUIDv8{
		Timestamp: 123456789,
		ClockSeq:  0x0800,
		Node:      []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
	}

	expectedUUID := uuidv8.ToString(uuid)

	mock.ExpectExec("INSERT INTO items").WithArgs(expectedUUID).WillReturnResult(sqlmock.NewResult(1, 1))

	_, err = db.Exec("INSERT INTO items (uuid) VALUES (?)", uuid)
	if err != nil {
		t.Errorf("Failed to execute mock database write: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestUUIDv8_Scan(t *testing.T) {
	// Mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	uuidStr := "9a3d4049-0e2c-8080-0102-030405060000"

	rows := sqlmock.NewRows([]string{"uuid"}).AddRow(uuidStr)
	mock.ExpectQuery("SELECT uuid FROM items").WillReturnRows(rows)

	var uuid uuidv8.UUIDv8
	err = db.QueryRow("SELECT uuid FROM items").Scan(&uuid)
	if err != nil {
		t.Errorf("Failed to scan mock database value: %v", err)
	}

	if uuidv8.ToString(&uuid) != uuidStr {
		t.Errorf("Expected UUIDv8 %s, got %s", uuidStr, uuidv8.ToString(&uuid))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}
