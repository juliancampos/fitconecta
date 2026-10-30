package payments

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuildPixPayload(t *testing.T) {
	payload := buildPixPayload(15000, "mock_abc123")

	if !strings.HasPrefix(payload, "000201") {
		t.Fatalf("payload nao inicia com o indicador EMV esperado: %q", payload[:8])
	}
	if !strings.Contains(payload, "5406150.00") {
		t.Fatalf("payload nao contem o valor esperado: %s", payload)
	}
	if !strings.Contains(payload, "5303986") {
		t.Fatalf("payload nao contem a moeda BRL: %s", payload)
	}
	if !strings.Contains(payload, "mockabc123") {
		t.Fatalf("payload nao contem o txid sanitizado: %s", payload)
	}

	expected := fmt.Sprintf("%04X", crc16(payload[:len(payload)-4]))
	if payload[len(payload)-4:] != expected {
		t.Fatalf("CRC invalido: esperado %s, obteve %s", expected, payload[len(payload)-4:])
	}
}

func TestBuildPixPayloadTruncatesTxID(t *testing.T) {
	txid := "abcdefghijklmnopqrstuvwxyz0123456789"
	payload := buildPixPayload(1000, txid)
	if !strings.Contains(payload, txid[:25]) {
		t.Fatalf("txid deveria ser truncado para 25 caracteres: %s", payload)
	}
	if strings.Contains(payload, txid) {
		t.Fatalf("txid nao deveria aparecer completo: %s", payload)
	}
}
