package router

import "testing"

func TestValidateSource_RequiredFieldsPerMode(t *testing.T) {
	tests := []struct {
		name    string
		body    sourceBody
		wantErr bool
	}{
		{"online valid", sourceBody{Source: "online", NamaBarang: "Laptop", LinkPembelian: "https://example.com/item"}, false},
		{"online link required", sourceBody{Source: "online", NamaBarang: "Laptop"}, true},
		{"offline valid", sourceBody{Source: "offline", NamaBarang: "ATK", NamaToko: "Toko A", NoTelp: "021123456", NoRekening: "123"}, false},
		{"offline fields required", sourceBody{Source: "offline", NamaBarang: "ATK", NamaToko: "Toko A"}, true},
		{"mode invalid", sourceBody{Source: "marketplace", NamaBarang: "ATK"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateSource(&tt.body); (got != nil) != tt.wantErr {
				t.Fatalf("validateSource() error = %v, wantErr %v", got, tt.wantErr)
			}
		})
	}
}
