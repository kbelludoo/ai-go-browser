package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// NavigateReady waits for this navigation's load lifecycle event, without a
// fixed settling delay. Dynamic applications still need an explicit selector
// or application condition after navigation. Do not call concurrently with a
// Browser live session, which owns its CDP event stream.
func (s *StealthSession) NavigateReady(ctx context.Context, targetURL string) error {
	if s == nil || s.pageCDP == nil {
		return errors.New("no browser session")
	}
	if _, err := s.pageCDP.send(ctx, "Page.setLifecycleEventsEnabled", map[string]any{"enabled": true}); err != nil {
		return err
	}
	// Discard old navigation events before sending this request.
	for {
		select {
		case <-s.pageCDP.events:
		default:
			goto drained
		}
	}
drained:
	raw, err := s.pageCDP.send(ctx, "Page.navigate", map[string]any{"url": targetURL})
	if err != nil {
		return err
	}
	var navigation struct {
		ErrorText string `json:"errorText"`
		LoaderID  string `json:"loaderId"`
	}
	if err := json.Unmarshal(raw, &navigation); err != nil {
		return err
	}
	if navigation.ErrorText != "" {
		return fmt.Errorf("navigate %s: %s", targetURL, navigation.ErrorText)
	}
	if navigation.LoaderID == "" {
		// Same-document navigation does not create a new loader.
		return nil
	}
	for {
		select {
		case event := <-s.pageCDP.events:
			if event.Method != "Page.lifecycleEvent" {
				continue
			}
			var lifecycle struct {
				Name     string `json:"name"`
				LoaderID string `json:"loaderId"`
			}
			if json.Unmarshal(event.Params, &lifecycle) == nil && lifecycle.LoaderID == navigation.LoaderID && lifecycle.Name == "load" {
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("navigation readiness: %w", ctx.Err())
		case <-s.pageCDP.closed:
			return fmt.Errorf("navigation connection closed: %v", s.pageCDP.readErr)
		}
	}
}
