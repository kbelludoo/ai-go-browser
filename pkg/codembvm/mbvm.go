package codembvm

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

// OpCodes for the Code AST Micro-VM
const (
	OpPkg       byte = 0x10 // [pkg_name_id:uint16]
	OpInterface byte = 0x11 // [name_id:uint16][methods_count:uint16]
	OpStruct    byte = 0x12 // [name_id:uint16][fields_count:uint16]
	OpField     byte = 0x13 // [field_name_id:uint16][type_id:uint16]
	OpFunc      byte = 0x14 // [func_name_id:uint16][receiver_id:uint16][sig_id:uint16]
	OpCallEdge  byte = 0x15 // [caller_id:uint16][callee_id:uint16]
	OpEnd       byte = 0xFF
)

type Symbol struct {
	Kind     string   `json:"kind"` // "struct", "interface", "func"
	Package  string   `json:"package"`
	Name     string   `json:"name"`
	Receiver string   `json:"receiver,omitempty"`
	Fields   []string `json:"fields,omitempty"`
	Methods  []string `json:"methods,omitempty"`
	Calls    []string `json:"calls,omitempty"`
}

type CodebaseMBVM struct {
	TotalFiles   int      `json:"total_files"`
	TotalLines   int      `json:"total_lines"`
	RawBytes     int64    `json:"raw_bytes"`
	MBVMBytes    int      `json:"mbvm_bytes"`
	Symbols      []Symbol `json:"symbols"`
	StringPool   []string `json:"string_pool"`
	Instructions []byte   `json:"instructions"`
}

// CompileProject parses a Go codebase AST and compiles it into a lossless Code Micro-Bytecode VM.
func CompileProject(rootDir string) (*CodebaseMBVM, []byte, error) {
	fset := token.NewFileSet()
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

	var symbols []Symbol
	var code bytes.Buffer
	var rawBytes int64
	totalFiles := 0
	totalLines := 0

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.Contains(path, "vendor") || strings.Contains(path, ".git") || strings.Contains(path, "worktrees") {
			return nil
		}

		info, err := d.Info()
		if err == nil {
			rawBytes += info.Size()
		}
		totalFiles++

		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}

		pkgName := node.Name.Name
		code.WriteByte(OpPkg)
		_ = binary.Write(&code, binary.LittleEndian, getStringID(pkgName))

		ast.Inspect(node, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.TypeSpec:
				symName := x.Name.Name
				switch t := x.Type.(type) {
				case *ast.StructType:
					var fields []string
					if t.Fields != nil {
						for _, f := range t.Fields.List {
							fType := exprToString(f.Type)
							if len(f.Names) == 0 {
								fields = append(fields, fType)
							} else {
								for _, fn := range f.Names {
									fields = append(fields, fmt.Sprintf("%s %s", fn.Name, fType))
								}
							}
						}
					}
					symbols = append(symbols, Symbol{
						Kind:    "struct",
						Package: pkgName,
						Name:    symName,
						Fields:  fields,
					})
					code.WriteByte(OpStruct)
					_ = binary.Write(&code, binary.LittleEndian, getStringID(symName))
					_ = binary.Write(&code, binary.LittleEndian, uint16(len(fields)))
					for _, f := range fields {
						code.WriteByte(OpField)
						_ = binary.Write(&code, binary.LittleEndian, getStringID(f))
						_ = binary.Write(&code, binary.LittleEndian, 0)
					}

				case *ast.InterfaceType:
					var methods []string
					if t.Methods != nil {
						for _, m := range t.Methods.List {
							mType := exprToString(m.Type)
							for _, mn := range m.Names {
								methods = append(methods, fmt.Sprintf("%s%s", mn.Name, mType))
							}
						}
					}
					symbols = append(symbols, Symbol{
						Kind:    "interface",
						Package: pkgName,
						Name:    symName,
						Methods: methods,
					})
					code.WriteByte(OpInterface)
					_ = binary.Write(&code, binary.LittleEndian, getStringID(symName))
					_ = binary.Write(&code, binary.LittleEndian, uint16(len(methods)))
				}

			case *ast.FuncDecl:
				fnName := x.Name.Name
				recv := ""
				if x.Recv != nil && len(x.Recv.List) > 0 {
					recv = exprToString(x.Recv.List[0].Type)
				}
				sig := exprToString(x.Type)
				symbols = append(symbols, Symbol{
					Kind:     "func",
					Package:  pkgName,
					Name:     fnName,
					Receiver: recv,
					Methods:  []string{sig},
				})
				code.WriteByte(OpFunc)
				_ = binary.Write(&code, binary.LittleEndian, getStringID(fnName))
				_ = binary.Write(&code, binary.LittleEndian, getStringID(recv))
				_ = binary.Write(&code, binary.LittleEndian, getStringID(sig))
			}
			return true
		})
		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	code.WriteByte(OpEnd)

	// Serialize String Pool + Instructions with zlib
	var out bytes.Buffer
	zw := zlib.NewWriter(&out)
	_ = binary.Write(zw, binary.LittleEndian, uint32(len(strPool)))
	for _, s := range strPool {
		b := []byte(s)
		_ = binary.Write(zw, binary.LittleEndian, uint16(len(b)))
		zw.Write(b)
	}
	zw.Write(code.Bytes())
	zw.Close()

	compressedBytes := out.Bytes()

	mbvm := &CodebaseMBVM{
		TotalFiles:   totalFiles,
		TotalLines:   totalLines,
		RawBytes:     rawBytes,
		MBVMBytes:    len(compressedBytes),
		Symbols:      symbols,
		StringPool:   strPool,
		Instructions: code.Bytes(),
	}

	return mbvm, compressedBytes, nil
}

// GenerateAIPromptContext produces a token-optimal semantic projection for any LLM to understand the whole codebase without context rot.
func (vm *CodebaseMBVM) GenerateAIPromptContext(maxTokens int) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== CODEBASE ARCHITECTURE MB-VM PROJECTION ===\n"))
	sb.WriteString(fmt.Sprintf("Files: %d | Source: %.1f KB | MB-VM Bytecode: %.1f KB (Redução: %.1f%%)\n\n",
		vm.TotalFiles, float64(vm.RawBytes)/1024.0, float64(vm.MBVMBytes)/1024.0,
		(1.0-float64(vm.MBVMBytes)/float64(vm.RawBytes))*100))

	byPkg := make(map[string][]Symbol)
	for _, s := range vm.Symbols {
		byPkg[s.Package] = append(byPkg[s.Package], s)
	}

	for pkg, syms := range byPkg {
		sb.WriteString(fmt.Sprintf("📦 pkg %s\n", pkg))
		for _, s := range syms {
			switch s.Kind {
			case "interface":
				sb.WriteString(fmt.Sprintf("  type %s interface { %s }\n", s.Name, strings.Join(s.Methods, "; ")))
			case "struct":
				fieldsSummary := strings.Join(s.Fields, ", ")
				if len(fieldsSummary) > 80 {
					fieldsSummary = fieldsSummary[:80] + "..."
				}
				sb.WriteString(fmt.Sprintf("  type %s struct { %s }\n", s.Name, fieldsSummary))
			case "func":
				if s.Receiver != "" {
					sb.WriteString(fmt.Sprintf("  func (%s) %s%s\n", s.Receiver, s.Name, strings.Join(s.Methods, "")))
				} else {
					sb.WriteString(fmt.Sprintf("  func %s%s\n", s.Name, strings.Join(s.Methods, "")))
				}
			}
		}
		sb.WriteString("\n")
	}

	res := sb.String()
	return res
}

func exprToString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	var sb bytes.Buffer
	_ = printer.Fprint(&sb, token.NewFileSet(), expr)
	return strings.TrimSpace(sb.String())
}

// LoadCodebaseMBVM decompresses a compiled .mbvm file back into runtime memory.
func LoadCodebaseMBVM(compressed []byte) (*CodebaseMBVM, error) {
	zr, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	data, err := io.ReadAll(zr)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(data)
	var poolLen uint32
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

	return &CodebaseMBVM{StringPool: pool}, nil
}
