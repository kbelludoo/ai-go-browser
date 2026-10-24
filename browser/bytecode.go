package browser

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
)

// OpCodes for the Web Knowledge Micro-VM
const (
	OpDocInfo byte = 0x01 // [status:uint16][title_str_id:uint16][url_str_id:uint16]
	OpHeading byte = 0x02 // [level:uint8][str_id:uint16]
	OpContent byte = 0x03 // [str_id:uint16]
	OpLink    byte = 0x04 // [label_id:uint16][href_id:uint16]
	OpEnd     byte = 0xFF
)

// DocumentKnowledge represents structured content for bytecode compilation.
type DocumentKnowledge struct {
	URL        string      `json:"url"`
	Title      string      `json:"title"`
	StatusCode int         `json:"status_code"`
	Headings   []string    `json:"headings"`
	Content    []string    `json:"content"`
	Links      [][2]string `json:"links"` // [label, href]
}

// CompileToBytecode compiles a document into a compressed, lossless micro-executable bytecode.
func CompileToBytecode(doc DocumentKnowledge) []byte {
	strMap := make(map[string]uint16)
	var strPool []string

	getStringID := func(s string) uint16 {
		if id, ok := strMap[s]; ok {
			return id
		}
		id := uint16(len(strPool))
		strMap[s] = id
		strPool = append(strPool, s)
		return id
	}

	var code bytes.Buffer

	// OpDocInfo
	code.WriteByte(OpDocInfo)
	_ = binary.Write(&code, binary.LittleEndian, uint16(doc.StatusCode))
	_ = binary.Write(&code, binary.LittleEndian, getStringID(doc.Title))
	_ = binary.Write(&code, binary.LittleEndian, getStringID(doc.URL))

	// Headings
	for _, h := range doc.Headings {
		code.WriteByte(OpHeading)
		code.WriteByte(2)
		_ = binary.Write(&code, binary.LittleEndian, getStringID(h))
	}

	// Content
	for _, c := range doc.Content {
		code.WriteByte(OpContent)
		_ = binary.Write(&code, binary.LittleEndian, getStringID(c))
	}

	// Links
	for _, l := range doc.Links {
		code.WriteByte(OpLink)
		_ = binary.Write(&code, binary.LittleEndian, getStringID(l[0]))
		_ = binary.Write(&code, binary.LittleEndian, getStringID(l[1]))
	}

	code.WriteByte(OpEnd)

	// Serialize string pool + bytecode with zlib compression for lossless micro-binary
	var out bytes.Buffer
	zw := zlib.NewWriter(&out)
	_ = binary.Write(zw, binary.LittleEndian, uint16(len(strPool)))
	for _, s := range strPool {
		b := []byte(s)
		_ = binary.Write(zw, binary.LittleEndian, uint16(len(b)))
		zw.Write(b)
	}
	zw.Write(code.Bytes())
	zw.Close()

	return out.Bytes()
}

// BytecodeVM is an in-memory runtime executing query instructions against the compiled bytecode.
type BytecodeVM struct {
	StringPool []string
	StatusCode int
	Title      string
	URL        string
	Headings   []string
	Content    []string
	Links      [][2]string
}

// LoadBytecode decompresses and executes the bytecode into a structured queryable VM.
func LoadBytecode(compressed []byte) (*BytecodeVM, error) {
	zr, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("invalid bytecode header: %w", err)
	}
	defer zr.Close()

	data, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("failed decompressing bytecode: %w", err)
	}

	r := bytes.NewReader(data)
	var poolLen uint16
	if err := binary.Read(r, binary.LittleEndian, &poolLen); err != nil {
		return nil, err
	}

	pool := make([]string, poolLen)
	for i := 0; i < int(poolLen); i++ {
		var strLen uint16
		_ = binary.Read(r, binary.LittleEndian, &strLen)
		strBytes := make([]byte, strLen)
		_, _ = r.Read(strBytes)
		pool[i] = string(strBytes)
	}

	vm := &BytecodeVM{StringPool: pool}
	for {
		op, err := r.ReadByte()
		if err != nil || op == OpEnd {
			break
		}
		switch op {
		case OpDocInfo:
			var status, titleID, urlID uint16
			_ = binary.Read(r, binary.LittleEndian, &status)
			_ = binary.Read(r, binary.LittleEndian, &titleID)
			_ = binary.Read(r, binary.LittleEndian, &urlID)
			vm.StatusCode = int(status)
			if int(titleID) < len(pool) {
				vm.Title = pool[titleID]
			}
			if int(urlID) < len(pool) {
				vm.URL = pool[urlID]
			}
		case OpHeading:
			_, _ = r.ReadByte()
			var hid uint16
			_ = binary.Read(r, binary.LittleEndian, &hid)
			if int(hid) < len(pool) {
				vm.Headings = append(vm.Headings, pool[hid])
			}
		case OpContent:
			var cid uint16
			_ = binary.Read(r, binary.LittleEndian, &cid)
			if int(cid) < len(pool) {
				vm.Content = append(vm.Content, pool[cid])
			}
		case OpLink:
			var lid, hid uint16
			_ = binary.Read(r, binary.LittleEndian, &lid)
			_ = binary.Read(r, binary.LittleEndian, &hid)
			if int(lid) < len(pool) && int(hid) < len(pool) {
				vm.Links = append(vm.Links, [2]string{pool[lid], pool[hid]})
			}
		}
	}
	return vm, nil
}
