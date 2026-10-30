package payments

import (
	"fmt"
	"strings"
)

// buildPixPayload monta um BR Code (EMV) sintaticamente valido para o
// provedor mock. A chave usada e ficticia e nao liquida pagamentos reais.
func buildPixPayload(amountCents int, txid string) string {
	txid = sanitizeTxID(txid)
	if len(txid) > 25 {
		txid = txid[:25]
	}

	merchantAccount := tlv("00", "br.gov.bcb.pix") +
		tlv("01", "pix@fitconecta.test") +
		tlv("02", "FitConecta mock - nao pagar")

	payload := tlv("00", "01") +
		tlv("26", merchantAccount) +
		tlv("52", "0000") +
		tlv("53", "986") +
		tlv("54", formatAmount(amountCents)) +
		tlv("58", "BR") +
		tlv("59", "FITCONECTA") +
		tlv("60", "SAO PAULO") +
		tlv("62", tlv("05", txid)) +
		"6304"

	return payload + fmt.Sprintf("%04X", crc16(payload))
}

func tlv(id, value string) string {
	return fmt.Sprintf("%s%02d%s", id, len(value), value)
}

func formatAmount(cents int) string {
	if cents < 0 {
		cents = 0
	}
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func sanitizeTxID(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func crc16(payload string) uint16 {
	var crc uint16 = 0xFFFF
	for i := 0; i < len(payload); i++ {
		crc ^= uint16(payload[i]) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
