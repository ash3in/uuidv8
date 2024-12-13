# uuidv8
[![Go Reference](https://pkg.go.dev/badge/github.com/ash3in/uuidv8.svg)](https://pkg.go.dev/github.com/ash3in/uuidv8)
[![Go Report Card](https://goreportcard.com/badge/github.com/ash3in/uuidv8)](https://goreportcard.com/report/github.com/ash3in/uuidv8)
[![Coverage Status](https://codecov.io/gh/ash3in/uuidv8/branch/main/graph/badge.svg)](https://codecov.io/gh/ash3in/uuidv8)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Your go-to for all things UUIDv8 in Go.

---

## Why this library?

Hey there! Welcome to **uuidv8**, a Go library built for developers who live and breathe distributed systems - whether you’re wrangling microservices, data pipelines or processing transaction ledgers in fintech. `UUIDv8` might be afresh on the block, but its flexibility may become a game-changer for time-based unique identifiers.

After years of building modern fintech systems where every millisecond counts, I found myself transitioning from `UUIDv4` to `UUIDv7` for its time-first structure. Then there's `UUIDv8` and… no big solid Go libraries to support it. So, I decided to take a step.

**uuidv8** is simple, clean and built with real-world use in mind. No bloat. No unnecessary dependencies. Just the Go standard library, doing what it does best. It’s perfect for scenarios where precision, reliability and ease of use matter - because let’s be honest, that’s most of our work.

---

## Highlights

- **Zero external dependencies**: Built entirely on Go’s standard library.
- **Real-world focus**: Designed with distributed systems and precision-critical workflows in mind.
- **Flexibility**: Use `New()` for simplicity or `NewWithParams()` when you need custom configurations.
- **Thoroughly tested**: Built and tested with the same rigor you’d expect in a production fintech system.

---

## Installation

Get started in seconds:

```bash
go get github.com/ash3in/uuidv8
```

That’s it. No extras, no setup headaches.

---

## Quick Start

### The Easy Way: `New()`

If all you need is a reliable UUIDv8, `New()` has you covered.

```go
package main

import (
	"fmt"
	"log"

	"github.com/ash3in/uuidv8"
)

func main() {
	uuid, err := uuidv8.New()
	if err != nil {
		log.Fatalf("Error generating UUIDv8: %v", err)
	}
	fmt.Println("Generated UUIDv8:", uuid)
}
```

No fuss. No setup. Just a fully compliant UUIDv8 - ready for your system.

---

### Full Control: `NewWithParams()`

Need more control? You can customize everything: timestamp, clock sequence, and node.

```go
timestamp := uint64(1633024800000000000) // Custom timestamp
clockSeq := uint16(1234)                // Custom clock sequence
node := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06} // Custom node

uuid, err := uuidv8.NewWithParams(timestamp, clockSeq, node, uuidv8.TimestampBits48)
if err != nil {
	log.Fatalf("Error generating custom UUIDv8: %v", err)
}
fmt.Println("Custom UUIDv8:", uuid)
```

Perfect for deterministic UUIDs in tests or tightly controlled distributed environments.

---

### Parse and Validate UUIDv8s

Easily parse UUIDv8 strings or validate their compliance:

```go
uuidStr := "01b69b4f-0000-8800-0102-030405060000"

// Parse UUIDv8
parsed, err := uuidv8.FromString(uuidStr)
if err != nil {
	log.Fatalf("Error parsing UUIDv8: %v", err)
}
fmt.Printf("Timestamp: %d, ClockSeq: %d, Node: %x\n", parsed.Timestamp, parsed.ClockSeq, parsed.Node)

// Validate UUIDv8
if uuidv8.IsValidUUIDv8(uuidStr) {
	fmt.Println("Valid UUIDv8")
} else {
	fmt.Println("Invalid UUIDv8")
}
```

---

### JSON Serialization and Deserialization

Seamlessly integrate UUIDv8 with your APIs and data storage:

```go
// Serialize
uuid := &uuidv8.UUIDv8{Timestamp: 123456789, ClockSeq: 0x0800, Node: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}}
data, _ := json.Marshal(uuid)
fmt.Println(string(data)) // Output: "01b69b4f-0000-8800-0102-030405060000"

// Deserialize
var parsedUUID uuidv8.UUIDv8
json.Unmarshal([]byte(`"01b69b4f-0000-8800-0102-030405060000"`), &parsedUUID)
fmt.Printf("Parsed UUIDv8: %+v\n", parsedUUID)
```

---

## Why UUIDv8?

UUIDv8 is designed for scenarios where flexibility and time-based uniqueness are critical. It bridges the gap between structure and freedom - ideal for event logs, transaction IDs, or any use case that demands precise identifiers. And with **uuidv8**, you get full compliance with the [spec](https://www.ietf.org/archive/id/draft-peabody-dispatch-new-uuid-format-01.html#name-uuidv8-layout-and-bit-order), minus the overhead.


## Who’s it for?

**Lightweight. Reliable. Built for Go Devs.**

If you're building fintech solutions or distributed applications and need UUIDv8 support that's robust yet lightweight, uuidv8 is for you. It's crafted to simplify your work while keeping your systems reliable

---

## Testing

Whether it's high-concurrency workloads in distributed systems or edge cases like all-zero UUIDs, this library has been tested to handle them all.

Run the tests yourself:

```bash
go test ./...
```

---

## Contributing

Got ideas? Found a bug or a mistake? Think this could be even better? Let’s make it happen. Open an issue or a PR and let’s collaborate.

---

## License

MIT License. Do whatever you want with it – just build something awesome.

