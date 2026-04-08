package mqtt

import (
	"encoding/json"
	"fmt"
	"strings"

	"iot-hub/backend/internal/model"
)

func ParseStatus(payload []byte) model.LEDStatusUpdate {
	update := model.LEDStatusUpdate{
		Kind:    "led_strip",
		RawJSON: string(payload),
	}

	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return update
	}

	if kind, ok := data["kind"].(string); ok && strings.TrimSpace(kind) != "" {
		update.Kind = strings.TrimSpace(kind)
	}

	if v, ok := data["power"].(bool); ok {
		update.Power = &v
	}
	if v, ok := data["brightness"].(float64); ok {
		n := int(v)
		update.Brightness = &n
	}
	if v, ok := data["color"].(string); ok && strings.TrimSpace(v) != "" {
		c := strings.TrimSpace(v)
		update.Color = &c
	}
	if v, ok := data["pixelPin"].(float64); ok {
		n := int(v)
		update.PixelPin = &n
	}

	return update
}

func BuildCommandPayload(cmd model.LEDCommand) ([]byte, error) {
	if cmd.Brightness != nil && (*cmd.Brightness < 0 || *cmd.Brightness > 255) {
		return nil, fmt.Errorf("brightness must be between 0 and 255")
	}
	if cmd.PixelPin != nil && *cmd.PixelPin < 0 {
		return nil, fmt.Errorf("pixelPin must be non-negative")
	}
	return json.Marshal(cmd)
}
