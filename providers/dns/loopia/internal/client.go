package internal

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.loopia.se/RPCSERV"

const (
	returnOk        = "OK"
	returnAuthError = "AUTH_ERROR"
)

// Client the Loopia client.
type Client struct {
	APIUser     string
	APIPassword string
	BaseURL     string
	HTTPClient  *http.Client
}

// NewClient creates a new Loopia Client.
func NewClient(apiUser, apiPassword string) *Client {
	return &Client{
		APIUser:     apiUser,
		APIPassword: apiPassword,
		BaseURL:     DefaultBaseURL,
		HTTPClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// AddTXTRecord adds a TXT record.
func (c *Client) AddTXTRecord(domain string, subdomain string, ttl int, value string) error {
	call := &methodCall{
		MethodName: "addZoneRecord",
		Params: []param{
			paramString{Value: c.APIUser},
			paramString{Value: c.APIPassword},
			paramString{Value: domain},
			paramString{Value: subdomain},
			paramStruct{
				StructMembers: []structMember{
					structMemberString{
						Name:  "type",
						Value: "TXT",
					}, structMemberInt{
						Name:  "ttl",
						Value: ttl,
					}, structMemberInt{
						Name:  "priority",
						Value: 0,
					}, structMemberString{
						Name:  "rdata",
						Value: value,
					}, structMemberInt{
						Name:  "record_id",
						Value: 0,
					},
				},
			},
		},
	}
	resp := &responseString{}

	err := c.rpcCall(call, resp)
	if err != nil {
		return err
	}

	switch v := strings.TrimSpace(resp.Value); v {
	case returnOk:
		return nil
	case returnAuthError:
		return errors.New("authentication error")
	default:
		return fmt.Errorf("unknown error: %q", v)
	}
}

// RemoveTXTRecord removes a TXT record.
func (c *Client) RemoveTXTRecord(domain string, subdomain string, recordID int) error {
	call := &methodCall{
		MethodName: "removeZoneRecord",
		Params: []param{
			paramString{Value: c.APIUser},
			paramString{Value: c.APIPassword},
			paramString{Value: domain},
			paramString{Value: subdomain},
			paramInt{Value: recordID},
		},
	}
	resp := &responseString{}

	err := c.rpcCall(call, resp)
	if err != nil {
		return err
	}

	switch v := strings.TrimSpace(resp.Value); v {
	case returnOk:
		return nil
	case returnAuthError:
		return fmt.Errorf("authentication error")
	default:
		return fmt.Errorf("unknown error: %q", v)
	}
}

// GetTXTRecords gets TXT records.
func (c *Client) GetTXTRecords(domain string, subdomain string) ([]RecordObj, error) {
	call := &methodCall{
		MethodName: "getZoneRecords",
		Params: []param{
			paramString{Value: c.APIUser},
			paramString{Value: c.APIPassword},
			paramString{Value: domain},
			paramString{Value: subdomain},
		},
	}
	resp := &recordObjectsResponse{}

	err := c.rpcCall(call, resp)

	return resp.Params, err
}

// RemoveSubdomain remove a sub-domain.
func (c *Client) RemoveSubdomain(domain, subdomain string) error {
	call := &methodCall{
		MethodName: "removeSubdomain",
		Params: []param{
			paramString{Value: c.APIUser},
			paramString{Value: c.APIPassword},
			paramString{Value: domain},
			paramString{Value: subdomain},
		},
	}
	resp := &responseString{}

	err := c.rpcCall(call, resp)
	if err != nil {
		return err
	}

	switch v := strings.TrimSpace(resp.Value); v {
	case returnOk:
		return nil
	case returnAuthError:
		return errors.New("authentication error")
	default:
		return fmt.Errorf("unknown error: %q", v)
	}
}

// rpcCall makes an XML-RPC call to Loopia's RPC endpoint
// by marshaling the data given in the call argument to XML and sending that via HTTP Post to Loopia.
// The response is then unmarshalled into the resp argument.
func (c *Client) rpcCall(call *methodCall, resp response) error {
	b, err := xml.MarshalIndent(call, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	b = append([]byte(`<?xml version="1.0"?>`+"\n"), b...)

	respBody, err := c.httpPost(c.BaseURL, "text/xml", bytes.NewReader(b))
	if err != nil {
		return err
	}

	err = xml.Unmarshal(respBody, resp)
	if err != nil {
		return fmt.Errorf("unmarshal error: %w", err)
	}

	if resp.faultCode() != 0 {
		return rpcError{
			faultCode:   resp.faultCode(),
			faultString: strings.TrimSpace(resp.faultString()),
		}
	}

	return nil
}

func (c *Client) httpPost(url string, bodyType string, body io.Reader) ([]byte, error) {
	resp, err := c.HTTPClient.Post(url, bodyType, body)
	if err != nil {
		return nil, fmt.Errorf("HTTP Post Error: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP Post Error: %d", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HTTP Post Error: %w", err)
	}

	return b, nil
}
