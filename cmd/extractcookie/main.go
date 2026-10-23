package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"ai-go-browser/pkg/session"
)

var (
	crypt32              = syscall.NewLazyDLL("crypt32.dll")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func cryptUnprotect(data []byte) ([]byte, error) {
	var inBlob, outBlob dataBlob
	inBlob.cbData = uint32(len(data))
	inBlob.pbData = &data[0]

	ret, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if ret == 0 {
		return nil, err
	}
	defer procCryptUnprotectData.Call(uintptr(unsafe.Pointer(outBlob.pbData)))
	out := make([]byte, outBlob.cbData)
	copy(out, unsafe.Slice(outBlob.pbData, outBlob.cbData))
	return out, nil
}

func main() {
	localAppData := os.Getenv("LOCALAPPDATA")
	localStatePath := filepath.Join(localAppData, "BraveSoftware", "Brave-Browser", "User Data", "Local State")
	cookiesPath := filepath.Join(localAppData, "BraveSoftware", "Brave-Browser", "User Data", "Default", "Network", "Cookies")

	// 1. Read master key
	stateBytes, err := os.ReadFile(localStatePath)
	if err != nil {
		fmt.Printf("ReadFile err: %v\n", err)
		return
	}
	var state struct {
		OSCrypt struct {
			EncryptedKey string `json:"encrypted_key"`
		} `json:"os_crypt"`
	}
	json.Unmarshal(stateBytes, &state)

	rawKey, _ := base64.StdEncoding.DecodeString(state.OSCrypt.EncryptedKey)
	masterKey, err := cryptUnprotect(rawKey[5:])
	if err != nil {
		fmt.Printf("CryptUnprotect err: %v\n", err)
		return
	}
	fmt.Printf("Master key: %d bytes\n", len(masterKey))

	// 2. Read raw Cookies database
	dbBytes, err := os.ReadFile(cookiesPath)
	if err != nil {
		fmt.Printf("ReadFile cookies err: %v\n", err)
		return
	}

	block, _ := aes.NewCipher(masterKey)
	gcm, _ := cipher.NewGCM(block)

	// Scan for 'arena-auth-prod-v1.0' in db
	targetName := []byte("arena-auth-prod-v1.0")
	idx := bytes.Index(dbBytes, targetName)
	if idx == -1 {
		fmt.Println("Target cookie not found in DB bytes")
		return
	}

	fmt.Printf("Found 'arena-auth-prod-v1.0' at offset %d\n", idx)

	// In Chromium SQLite, encrypted_value follows within ~200 bytes and starts with 'v10'
	searchChunk := dbBytes[idx : idx+6000]
	v10Idx := bytes.Index(searchChunk, []byte("v10"))
	if v10Idx == -1 {
		fmt.Println("v10 prefix not found near target")
		return
	}

	encVal := searchChunk[v10Idx:]
	nonce := encVal[3:15]

	// Try decrypting with various lengths (the cookie length is typically 2000-4000 bytes)
	var authCookie string
	for l := 500; l < len(encVal)-15 && l < 5000; l++ {
		ciphertext := encVal[15:l]
		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err == nil && bytes.HasPrefix(plaintext, []byte("base64-")) {
			authCookie = string(plaintext)
			fmt.Printf("Decrypted auth cookie! Length = %d\n", len(authCookie))
			break
		}
	}

	if authCookie == "" {
		fmt.Println("Failed to match valid AES-GCM ciphertext length")
		return
	}

	// Verify
	req, _ := http.NewRequest("GET", "https://arena.ai/api/me", nil)
	req.Header.Set("Cookie", "arena-auth-prod-v1.0="+authCookie)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("API err: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("API Response status %d: %s\n", resp.StatusCode, string(body))

	// Save to session store
	store, _ := session.NewStore("", "")
	cookies := []*http.Cookie{
		{Name: "arena-auth-prod-v1.0", Value: authCookie, Domain: ".arena.ai", Path: "/"},
	}
	store.SaveCookies("arena.ai", cookies)
	fmt.Println("Saved to session.Store!")
}
