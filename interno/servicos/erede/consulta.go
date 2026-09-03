package erede

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type authorizationBlock struct {
	TID        string `json:"tid"`
	Reference  string `json:"reference"`
	ReturnCode string `json:"returnCode"`
	Status     string `json:"status"`
	Kind       string `json:"kind"`
}

type refundBlock struct {
	Status string `json:"status"`
	Amount string `json:"amount"`
}

type transactionResponse struct {
	TID           string              `json:"tid"`
	Reference     string              `json:"reference"`
	ReturnCode    string              `json:"returnCode"`
	Status        string              `json:"status"`
	Kind          string              `json:"kind"`
	Authorization *authorizationBlock `json:"authorization"`
	Refunds       []refundBlock       `json:"refunds"`
}

func (tx *transactionResponse) blocoPix() *authorizationBlock {
	if tx == nil {
		return nil
	}
	if tx.Authorization != nil {
		return tx.Authorization
	}
	if strings.TrimSpace(tx.TID) != "" || strings.TrimSpace(tx.Status) != "" || strings.TrimSpace(tx.ReturnCode) != "" {
		return &authorizationBlock{
			TID:        tx.TID,
			Reference:  tx.Reference,
			ReturnCode: tx.ReturnCode,
			Status:     tx.Status,
			Kind:       tx.Kind,
		}
	}
	return nil
}

func statusPixNormalizado(tx *transactionResponse) string {
	b := tx.blocoPix()
	if b == nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(b.Status))
}

// StatusPixLabel status legível para logs (ex.: Approved, Pending, Canceled).
func StatusPixLabel(tx *transactionResponse) string {
	b := tx.blocoPix()
	if b == nil {
		return ""
	}
	return strings.TrimSpace(b.Status)
}

// TransacaoCanceladaPix devolução total (status Canceled).
func TransacaoCanceladaPix(tx *transactionResponse) bool {
	s := statusPixNormalizado(tx)
	return s == "CANCELED" || s == "CANCELLED"
}

// TransacaoDevolucaoParcialPix Approved com refunds (ainda não zera o saldo).
func TransacaoDevolucaoParcialPix(tx *transactionResponse) bool {
	if tx == nil || TransacaoCanceladaPix(tx) || !TransacaoAprovadaPix(tx) {
		return false
	}
	return len(tx.Refunds) > 0
}

// ConsultarTransacaoPorTID GET /v2/transactions/{tid}
func ConsultarTransacaoPorTID(ctx context.Context, pv, clientSecret, ambiente, tid string) (*transactionResponse, error) {
	tid = strings.TrimSpace(tid)
	if tid == "" {
		return nil, fmt.Errorf("tid vazio")
	}
	bearer, err := ObterBearerToken(ctx, pv, clientSecret, ambiente)
	if err != nil {
		return nil, err
	}
	url := txURL(ambiente) + "/" + tid
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+bearer)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("e.rede consulta status %d: %s", res.StatusCode, strings.TrimSpace(string(b)))
	}
	var out transactionResponse
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// TransacaoAprovadaPix PIX pago: status Approved/Paid/Confirmed (ou returnCode 00 sem cancelamento).
func TransacaoAprovadaPix(tx *transactionResponse) bool {
	if TransacaoCanceladaPix(tx) {
		return false
	}
	b := tx.blocoPix()
	if b == nil {
		return false
	}
	s := strings.ToUpper(strings.TrimSpace(b.Status))
	if s == "APPROVED" || s == "PAID" || s == "CONFIRMED" {
		return true
	}
	// returnCode 00 só conta se o status não indicar cancelamento/negação.
	if strings.TrimSpace(b.ReturnCode) == "00" {
		if s == "" || s == "PENDING" {
			return true
		}
		if s == "DENIED" || s == "REJECTED" {
			return false
		}
		return true
	}
	return false
}
