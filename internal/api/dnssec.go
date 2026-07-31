package api

import "fmt"

func (c *Client) GetDNSSEC(zoneID string) (*DNSSECStatus, error) {
	var status DNSSECStatus
	err := c.do("GET", fmt.Sprintf("/zones/%s/dnssec", zoneID), nil, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

func (c *Client) EnableDNSSEC(zoneID string) (*DNSSECStatus, error) {
	var status DNSSECStatus
	err := c.do("POST", fmt.Sprintf("/zones/%s/dnssec/enable", zoneID), nil, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

func (c *Client) DisableDNSSEC(zoneID string) error {
	return c.do("POST", fmt.Sprintf("/zones/%s/dnssec/disable", zoneID), nil, nil)
}
