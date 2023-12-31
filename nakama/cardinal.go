package main

// cardinal.go wraps the http requests to some cardinal endpoints.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/heroiclabs/nakama-common/runtime"
	"pkg.world.dev/world-engine/sign"
)

var (
	listTxEndpointsEndpoint     = "list/tx-endpoints"
	listReadEndpoints           = "list/read-endpoints"
	createPersonaEndpoint       = "tx-create-persona"
	readPersonaSignerEndpoint   = "read-persona-signer"
	transactionReceiptsEndpoint = "transaction-receipts"

	readPersonaSignerStatusUnknown   = "unknown"
	readPersonaSignerStatusAvailable = "available"
	readPersonaSignerStatusAssigned  = "assigned"

	globalCardinalAddress string

	ErrorPersonaSignerAvailable = errors.New("persona signer is available")
	ErrorPersonaSignerUnknown   = errors.New("persona signer is unknown.")
)

type txResponse struct {
	TxHash string `json:"tx_hash"`
	Tick   uint64 `json:"tick"`
}

func initCardinalAddress() error {
	globalCardinalAddress = os.Getenv(EnvCardinalAddr)
	if globalCardinalAddress == "" {
		return fmt.Errorf("must specify a cardinal server via %s", EnvCardinalAddr)
	}
	return nil
}

func makeURL(resource string) string {
	return fmt.Sprintf("%s/%s", globalCardinalAddress, resource)
}

func cardinalListEndpoints(path string) ([]string, error) {
	url := makeURL(path)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		buf, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list endpoints (at %q) failed with status code %d: %v", url, resp.StatusCode, string(buf))
	}
	dec := json.NewDecoder(resp.Body)
	var endpoints []string
	if err := dec.Decode(&endpoints); err != nil {
		return nil, err
	}
	return endpoints, nil

}

func cardinalListAllEndpoints() ([]string, error) {
	var endpoints []string
	txs, err := cardinalListEndpoints(listTxEndpointsEndpoint)
	if err != nil {
		return nil, err
	}
	endpoints = append(endpoints, txs...)
	reads, err := cardinalListEndpoints(listReadEndpoints)
	if err != nil {
		return nil, err
	}
	endpoints = append(endpoints, reads...)
	return endpoints, nil
}

func doRequest(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %q failed: %w", req.URL, err)
	} else if resp.StatusCode != 200 {
		buf, err := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("got response of %d: %v, %w", resp.StatusCode, string(buf), err)
	}
	return resp, nil
}

func cardinalCreatePersona(ctx context.Context, nk runtime.NakamaModule, personaTag string) (txHash string, tick uint64, err error) {
	signerAddress := getSignerAddress()
	createPersonaTx := struct {
		PersonaTag    string
		SignerAddress string
	}{
		PersonaTag:    personaTag,
		SignerAddress: signerAddress,
	}

	key, nonce, err := getPrivateKeyAndANonce(ctx, nk)
	if err != nil {
		return "", 0, fmt.Errorf("unable to get the private key or a nonce: %w", err)
	}

	signedPayload, err := sign.NewSystemSignedPayload(key, globalNamespace, nonce, createPersonaTx)
	if err != nil {
		return "", 0, fmt.Errorf("unable to create signed payload: %w", err)
	}

	buf, err := signedPayload.Marshal()
	if err != nil {
		return "", 0, fmt.Errorf("unable to marshal signed payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", makeURL(createPersonaEndpoint), bytes.NewReader(buf))
	if err != nil {
		return "", 0, fmt.Errorf("unable to make request to %q: %w", createPersonaEndpoint, err)
	}

	resp, err := doRequest(req)
	if err != nil {
		return "", 0, err
	}
	if code := resp.StatusCode; code != 200 {
		buf, err := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("create persona response is not 200. code %v, body: %v, err: %v", code, string(buf), err)

	}
	var createPersonaResponse txResponse

	if err := json.NewDecoder(resp.Body).Decode(&createPersonaResponse); err != nil {
		return "", 0, fmt.Errorf("unable to decode response: %w", err)
	}
	if createPersonaResponse.TxHash == "" {
		return "", 0, fmt.Errorf("tx response does not have a tx hash")
	}
	return createPersonaResponse.TxHash, createPersonaResponse.Tick, nil
}

func cardinalQueryPersonaSigner(ctx context.Context, personaTag string, tick uint64) (signerAddress string, err error) {
	readPersonaRequest := struct {
		PersonaTag string
		Tick       uint64
	}{
		PersonaTag: personaTag,
		Tick:       tick,
	}

	buf, err := json.Marshal(readPersonaRequest)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", makeURL(readPersonaSignerEndpoint), bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	httpResp, err := doRequest(httpReq)
	if err != nil {
		return "", err
	}

	var resp struct {
		Status        string
		SignerAddress string
	}
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return "", err
	}
	if resp.Status == readPersonaSignerStatusUnknown {
		return "", ErrorPersonaSignerUnknown
	} else if resp.Status == readPersonaSignerStatusAvailable {
		return "", ErrorPersonaSignerAvailable
	}
	return resp.SignerAddress, nil
}
