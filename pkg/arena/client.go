package arena

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

// Client representa o cliente Go para consumir a API do Arena.
type Client struct {
	httpClient *http.Client
	baseURL    string
	user       *User
}

// User representa os dados do usuário autenticado.
type User struct {
	ID             string `json:"id"`
	Email          string `json:"email"`
	Username       string `json:"username"`
	EmailConfirmed bool   `json:"emailConfirmed"`
}

// Model representa um modelo no catálogo do Arena.
type Model struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	PublicName   string `json:"publicName"`
	Organization string `json:"organization"`
}

// NewClient cria um novo cliente Arena configurado com cookiejar.
func NewClient() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 45 * time.Second,
		},
		baseURL: "https://arena.ai",
	}
}

// streamClient compartilha jar/transport com o client principal, mas sem
// Timeout global — o SSE do create-evaluation pode durar minutos e o
// Timeout do http.Client corta a leitura do body inteiro.
func (c *Client) streamClient() *http.Client {
	return &http.Client{
		Jar:           c.httpClient.Jar,
		Transport:     c.httpClient.Transport,
		CheckRedirect: c.httpClient.CheckRedirect,
		Timeout:       0,
	}
}

// Login realiza a autenticação com e-mail e senha.
func (c *Client) Login(email, password string) (*User, error) {
	payload := map[string]any{
		"email":             email,
		"password":          password,
		"shouldLinkHistory": false,
	}
	bodyJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.baseURL+"/nextjs-api/sign-in/email", bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", c.baseURL)
	req.Header.Set("Referer", c.baseURL+"/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Error   string `json:"error"`
		User    *User  `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("falha ao decodificar resposta de login (status %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusOK || !res.Success {
		return nil, fmt.Errorf("login rejeitado: %s", res.Error)
	}

	c.user = res.User
	return res.User, nil
}

// GetUser obtém os detalhes da sessão atual (/api/me).
func (c *Client) GetUser() (*User, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sessão inválida ou não autenticada (status %d)", resp.StatusCode)
	}

	var res struct {
		User *User `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	c.user = res.User
	return res.User, nil
}

// GetModelCatalog retorna o catálogo de modelos disponíveis para benchmark/chat.
func (c *Client) GetModelCatalog() ([]Model, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/nextjs-api/model-catalog", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var arenas []struct {
		Arena  string  `json:"arena"`
		Models []Model `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&arenas); err != nil {
		return nil, err
	}

	for _, a := range arenas {
		if a.Arena == "text" || a.Arena == "chat" {
			return a.Models, nil
		}
	}
	if len(arenas) > 0 {
		return arenas[0].Models, nil
	}
	return nil, errors.New("nenhum modelo encontrado no catálogo")
}

// NewUUIDv7 gera um UUIDv7 compatível com a validação estrita do Arena.
func NewUUIDv7() string {
	var value [16]byte
	ms := uint64(time.Now().UnixMilli())
	value[0] = byte(ms >> 40)
	value[1] = byte(ms >> 32)
	value[2] = byte(ms >> 24)
	value[3] = byte(ms >> 16)
	value[4] = byte(ms >> 8)
	value[5] = byte(ms)
	_, _ = rand.Read(value[6:])
	value[6] = (value[6] & 0x0f) | 0x70
	value[8] = (value[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(value[0:4]),
		binary.BigEndian.Uint16(value[4:6]),
		binary.BigEndian.Uint16(value[6:8]),
		binary.BigEndian.Uint16(value[8:10]),
		value[10:16],
	)
}

// ChatRequest define a requisição de avaliação/chat.
type ChatRequest struct {
	Prompt           string
	Mode             string // "battle", "side-by-side", "direct"
	Modality         string // "chat", "auto"
	RecaptchaV3Token string
}

// ChatResponse contém o stream ou resposta unificada.
type ChatResponse struct {
	EvaluationID string
	Body         io.ReadCloser
}

// CreateEvaluation inicia uma sessão de avaliação/pergunta para as IAs no Arena.
func (c *Client) CreateEvaluation(req ChatRequest) (*ChatResponse, error) {
	if req.Mode == "" {
		req.Mode = "battle"
	}
	if req.Modality == "" {
		req.Modality = "chat"
	}

	evalID := NewUUIDv7()
	payload := map[string]any{
		"id":              evalID,
		"mode":            req.Mode,
		"modality":        req.Modality,
		"userMessageId":   NewUUIDv7(),
		"modelAMessageId": NewUUIDv7(),
		"modelBMessageId": NewUUIDv7(),
		"userMessage": map[string]any{
			"content": req.Prompt,
		},
	}
	if req.RecaptchaV3Token != "" {
		payload["recaptchaV3Token"] = req.RecaptchaV3Token
	}

	bodyJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/nextjs-api/stream/create-evaluation", bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream, application/json, */*")
	httpReq.Header.Set("Origin", c.baseURL)
	httpReq.Header.Set("Referer", c.baseURL+"/")
	httpReq.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := c.streamClient().Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API erro %d: %s", resp.StatusCode, string(respBody))
	}

	return &ChatResponse{
		EvaluationID: evalID,
		Body:         resp.Body,
	}, nil
}
