package researchclaw

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/morpheumstreet/CloseLoopAutomous/arms/internal/domain"
)

const maxInvokeResponseBytes = 8 << 20

// InvokeAllowlisted forwards one HTTP request to the hub using stored credentials.
// method is GET or POST; path must pass AllowedInvokePath; jsonBody is used only for POST.
func InvokeAllowlisted(ctx context.Context, hc *http.Client, hub *domain.ResearchHub, method, path string, jsonBody []byte) (statusCode int, respBody []byte, err error) {
	if hub == nil {
		return 0, nil, fmt.Errorf("researchclaw: nil hub")
	}
	m := strings.ToUpper(strings.TrimSpace(method))
	if !AllowedInvokePath(m, path) {
		return 0, nil, fmt.Errorf("researchclaw: path not allowed")
	}
	rawURL, err := joinAPI(hub.BaseURL, path)
	if err != nil {
		return 0, nil, err
	}
	if hc == nil {
		hc = http.DefaultClient
	}
	var body io.Reader
	if m == http.MethodPost && len(jsonBody) > 0 {
		body = bytes.NewReader(jsonBody)
	}
	req, err := http.NewRequestWithContext(ctx, m, rawURL, body)
	if err != nil {
		return 0, nil, err
	}
	if h := authHeader(hub.APIKey); h != "" {
		req.Header.Set("Authorization", h)
	}
	req.Header.Set("Accept", "application/json")
	if m == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := hc.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, maxInvokeResponseBytes))
	if err != nil {
		return res.StatusCode, nil, err
	}
	return res.StatusCode, b, nil
}
