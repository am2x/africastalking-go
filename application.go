package africastalking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/am2x/africastalking-go/internal/transport"
)

// ApplicationDataResp is a map which contains the application data.
type ApplicationDataResp struct {
	UserData Balance `json:"userData"`
}

// Balance is your africa's talking application balance. The format of this string is (3-digit Currency Code)(space)(Decimal Value) e.g KES 1785.50
type Balance struct {
	Balance string `json:"balance"`
}

// ApplicationData is a method to fetch user account balance from Africa's Talking API.
func (at *AfricasTalking) ApplicationData(ctx context.Context) (ApplicationDataResp, error) {
	resp, err := at.client.Do(ctx, transport.Request{}, at.apiKey)
	if err != nil {
		return ApplicationDataResp{}, fmt.Errorf("africastalking: application data: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return ApplicationDataResp{}, newAPIError(smsPath, resp)
	}
	var data ApplicationDataResp
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		return ApplicationDataResp{}, fmt.Errorf("africastalking: application data response: %w", err)
	}

	return data, nil
}
