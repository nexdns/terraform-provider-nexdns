package api

import "fmt"

func (c *Client) GetRecord(zoneID, recordID string) (*Record, error) {
	var record Record
	err := c.do("GET", fmt.Sprintf("/zones/%s/records/%s", zoneID, recordID), nil, &record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (c *Client) CreateRecord(zoneID string, req CreateRecordRequest) (*Record, error) {
	var record Record
	err := c.do("POST", fmt.Sprintf("/zones/%s/records", zoneID), req, &record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (c *Client) UpdateRecord(zoneID, recordID string, req UpdateRecordRequest) (*Record, error) {
	var record Record
	err := c.do("PUT", fmt.Sprintf("/zones/%s/records/%s", zoneID, recordID), req, &record)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (c *Client) DeleteRecord(zoneID, recordID string) error {
	return c.do("DELETE", fmt.Sprintf("/zones/%s/records/%s", zoneID, recordID), nil, nil)
}
