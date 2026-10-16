// Package ai/provider menyediakan klien HTTP untuk penyedia AI yang kompatibel
// dengan OpenAI (chat completions), termasuk smartsproxy.clouad/v1.
//
// Model yang didukung (sesuai dokumentasi penyedia):
//
//	gemini-3.8-flash-high
//	qwen/qwen3.8-omni-flash   (omni: teks + gambar, context 200M+)
package ai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ChatMessage adalah satu pesan dalam format OpenAI chat completions.
type ChatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string atau []ChatContentPart
}

// ChatContentPart adalah satu bagian konten (teks atau gambar).
type ChatContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL mengandung URL atau data URI gambar.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // low | high | auto
}

// ChatCompletionRequest adalah body request ke /chat/completions.
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	// ResponseFormat untuk memaksa JSON (jika didukung penyedia).
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

// ResponseFormat menentukan format respons.
type ResponseFormat struct {
	Type string `json:"type"` // json_object | text
}

// ChatCompletionResponse adalah respons dari /chat/completions.
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    int    `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

// OpenAIProvider adalah implementasi Provider melalui HTTP ke endpoint
// OpenAI-compatible.
type OpenAIProvider struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewOpenAIProvider membuat provider baru.
//
// baseURL harus sudah diakhiri tanpa trailing slash (mis.
// https://smartsproxy.clouad/v1). Endpoint chat completions akan
// ditambahkan sebagai /chat/completions.
func NewOpenAIProvider(baseURL, apiKey, model string, timeout time.Duration) *OpenAIProvider {
	return &OpenAIProvider{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ExtractReceipt mengirim gambar ke model AI dan mengembalikan hasil ekstraksi.
func (p *OpenAIProvider) ExtractReceipt(ctx context.Context, fileName string, mimeType string, data []byte) (*ExtractedReceipt, error) {
	if p.apiKey == "" {
		return nil, ErrProviderNotConfigured
	}

	// Encode gambar sebagai base64 data URI.
	b64 := base64.StdEncoding.EncodeToString(data)
	imageURL := fmt.Sprintf("data:%s;base64,%s", mimeType, b64)

	systemPrompt := `Anda adalah pakar parsing struk dan faktur pembelian. Ekstrak data terstruktur dari gambar struk/faktur yang diberikan.

Kembalikan HANYA objek JSON dengan field berikut (gunakan null atau string kosong kalau tidak terlihat):
- merchant_name: nama toko/merchant
- merchant_address: alamat toko
- merchant_phone: nomor telepon toko
- invoice_number: nomor faktur/invoice
- invoice_date: tanggal faktur (YYYY-MM-DD)
- purchase_date: tanggal pembelian (YYYY-MM-DD)
- payment_method: metode pembayaran (tunai, kartu kredit, debit, transfer, e-wallet, dst.)
- currency: mata uang (IDR, USD, dst.)
- subtotal: jumlah sebelum pajak (angka)
- tax: pajak (angka)
- discount: diskon (angka)
- total: total akhir (angka)
- items: array of { name, quantity, unit_price, total_price, unit }
- raw_text: teks utuh hasil OCR dari struk
- confidence: tingkat kepercayaan 0.0 - 1.0

Pastikan semua angka adalah tipe numerik (bukan string). Tanggal gunakan format YYYY-MM-DD.`

	userContent := []ChatContentPart{
		{Type: "text", Text: "Ekstrak data dari struk/faktur ini. Kembalikan dalam format JSON."},
		{Type: "image_url", ImageURL: &ImageURL{URL: imageURL}},
	}

	reqBody := ChatCompletionRequest{
		Model:          p.model,
		Temperature:    0.1,
		ResponseFormat: &ResponseFormat{Type: "json_object"},
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshall request AI: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("buat request AI: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("waktu permintaan ke AI habis: %w", err)
		}
		return nil, fmt.Errorf("permintaan ke AI gagal: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("baca respons AI: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("AI merespons status %d: %s", resp.StatusCode, string(body))
	}

	var parsed ChatCompletionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse respons AI: %w", err)
	}

	if parsed.Error != nil {
		return nil, fmt.Errorf("AI error: %s", parsed.Error.Message)
	}

	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("AI tidak mengembalikan pilihan")
	}

	content := parsed.Choices[0].Message.Content
	if content == "" {
		return nil, fmt.Errorf("AI mengembalikan konten kosong")
	}

	jsonText, err := parseJSONFromText(content)
	if err != nil {
		return nil, fmt.Errorf("ekstrak JSON dari respons AI: %w", err)
	}

	var rawMap map[string]any
	if err := json.Unmarshal([]byte(jsonText), &rawMap); err != nil {
		return nil, fmt.Errorf("parse JSON hasil ekstraksi: %w", err)
	}

	return normalizeReceipt(rawMap), nil
}

// Pastikan OpenAIProvider mengimplementasikan Provider.
var _ Provider = (*OpenAIProvider)(nil)
