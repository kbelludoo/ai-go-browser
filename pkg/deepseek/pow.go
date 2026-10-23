package deepseek

// Proof-of-work solver for DeepSeek's chat completion endpoint.
//
// DeepSeek gates POST /api/v0/chat/completion behind a proof-of-work header
// (x-ds-pow-response). The PoW algorithm ("DeepSeekHashV1") ships as a
// WebAssembly module the website loads from its own CDN
// (fe-static.deepseek.com/.../sha3_wasm_bg.wasm). Rather than reimplement its
// exact float64 hashing logic, we run DeepSeek's own module — the same bytecode
// the browser runs — inside the wazero sandbox (pure Go, no CGO, no ambient
// file/network access).

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

//go:embed sha3_wasm_bg.wasm
var sha3Wasm []byte

// PowChallenge is the biz_data.challenge object returned by
// POST /api/v0/chat/create_pow_challenge.
type PowChallenge struct {
	Algorithm  string  `json:"algorithm"`
	Challenge  string  `json:"challenge"`
	Salt       string  `json:"salt"`
	ExpireAt   int64   `json:"expire_at"`
	Difficulty float64 `json:"difficulty"`
	Signature  string  `json:"signature"`
	TargetPath string  `json:"target_path"`
}

// PowSolver runs DeepSeek's own WebAssembly PoW module. A single solver owns a
// wazero module instance which is not reentrant, so calls are serialized.
type PowSolver struct {
	mu      sync.Mutex
	rt      wazero.Runtime
	mod     api.Module
	memory  api.Memory
	malloc  api.Function
	stack   api.Function
	solveFn api.Function
}

// NewPowSolver compiles and instantiates the embedded PoW module once.
func NewPowSolver(ctx context.Context) (*PowSolver, error) {
	rt := wazero.NewRuntime(ctx)
	mod, err := rt.InstantiateWithConfig(ctx, sha3Wasm, wazero.NewModuleConfig())
	if err != nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("instantiate pow wasm: %w", err)
	}
	s := &PowSolver{
		rt:     rt,
		mod:    mod,
		memory: mod.Memory(),
	}
	for name, dst := range map[string]*api.Function{
		"__wbindgen_export_0":             &s.malloc,
		"__wbindgen_add_to_stack_pointer": &s.stack,
		"wasm_solve":                      &s.solveFn,
	} {
		fn := mod.ExportedFunction(name)
		if fn == nil {
			_ = rt.Close(ctx)
			return nil, fmt.Errorf("pow wasm missing export %q", name)
		}
		*dst = fn
	}
	if s.memory == nil {
		_ = rt.Close(ctx)
		return nil, fmt.Errorf("pow wasm missing exported memory")
	}
	return s, nil
}

// Close releases the wazero runtime and its module instance.
func (s *PowSolver) Close() error {
	if s == nil || s.rt == nil {
		return nil
	}
	return s.rt.Close(context.Background())
}

// writeString mallocs and copies a UTF-8 string into wasm memory.
func (s *PowSolver) writeString(ctx context.Context, text string) (uint32, uint32, error) {
	data := []byte(text)
	res, err := s.malloc.Call(ctx, uint64(len(data)), 1)
	if err != nil {
		return 0, 0, fmt.Errorf("pow malloc: %w", err)
	}
	ptr := uint32(res[0])
	if len(data) > 0 && !s.memory.Write(ptr, data) {
		return 0, 0, fmt.Errorf("pow memory write out of range at %d", ptr)
	}
	return ptr, uint32(len(data)), nil
}

// solve returns the integer PoW answer, or nil when the module reports failure
// (an expired challenge, for instance).
//
// Mirrors the website's wasm-bindgen call:
//
//	wasm_solve(retptr, challenge_ptr, challenge_len,
//	           prefix_ptr, prefix_len, difficulty)
//
// with a 16-byte return slot reserved on the shadow stack. The slot holds an
// i32 status flag at +0 and an f64 answer at +8.
func (s *PowSolver) solve(ctx context.Context, challenge, prefix string, difficulty float64) (*int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.stack.Call(ctx, api.EncodeI32(-16))
	if err != nil {
		return nil, fmt.Errorf("pow add_to_stack: %w", err)
	}
	retptr := uint32(api.DecodeI32(res[0]))
	defer func() { _, _ = s.stack.Call(ctx, api.EncodeI32(16)) }()

	cPtr, cLen, err := s.writeString(ctx, challenge)
	if err != nil {
		return nil, err
	}
	pPtr, pLen, err := s.writeString(ctx, prefix)
	if err != nil {
		return nil, err
	}

	if _, err = s.solveFn.Call(ctx, uint64(retptr), uint64(cPtr), uint64(cLen), uint64(pPtr), uint64(pLen), api.EncodeF64(difficulty)); err != nil {
		return nil, fmt.Errorf("pow wasm_solve: %w", err)
	}

	slot, ok := s.memory.Read(retptr, 16)
	if !ok {
		return nil, fmt.Errorf("pow read return slot out of range at %d", retptr)
	}
	status := int32(binary.LittleEndian.Uint32(slot[0:4]))
	if status == 0 {
		return nil, nil
	}
	value := math.Float64frombits(binary.LittleEndian.Uint64(slot[8:16]))
	answer := int64(value)
	return &answer, nil
}

// MakeHeader builds the base64 x-ds-pow-response value from a challenge.
func (s *PowSolver) MakeHeader(ctx context.Context, ch PowChallenge) (string, error) {
	prefix := fmt.Sprintf("%s_%d_", ch.Salt, ch.ExpireAt)
	answer, err := s.solve(ctx, ch.Challenge, prefix, ch.Difficulty)
	if err != nil {
		return "", err
	}
	if answer == nil {
		return "", fmt.Errorf("pow solver returned no answer (challenge expired?)")
	}
	payload := struct {
		Algorithm  string `json:"algorithm"`
		Challenge  string `json:"challenge"`
		Salt       string `json:"salt"`
		Answer     int64  `json:"answer"`
		Signature  string `json:"signature"`
		TargetPath string `json:"target_path"`
	}{
		Algorithm:  ch.Algorithm,
		Challenge:  ch.Challenge,
		Salt:       ch.Salt,
		Answer:     *answer,
		Signature:  ch.Signature,
		TargetPath: ch.TargetPath,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}
