package model

type Device struct {
	ID         int64     `json:"id"`
	MAC        string    `json:"mac"`
	Name       string    `json:"name"`
	RoomID     int64     `json:"roomId"`
	RoomName   string    `json:"roomName"`
	Kind       string    `json:"kind"`
	LastSeen   string    `json:"lastSeen"`
	StatusJSON string    `json:"statusJson"`
	LEDStrip   *LEDStrip `json:"ledStrip,omitempty"`
}

type LEDStrip struct {
	Power      bool   `json:"power"`
	Brightness int    `json:"brightness"`
	Color      string `json:"color"`
	PixelPin   int    `json:"pixelPin"`
	PixelCount int    `json:"pixelCount"`
}

type Room struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type DevicePatch struct {
	Name   *string `json:"name"`
	RoomID *int64  `json:"roomId"`
}

type LEDCommand struct {
	Power      *bool   `json:"power"`
	Brightness *int    `json:"brightness"`
	Color      *string `json:"color"`
	PixelPin   *int    `json:"pixelPin"`
}

type LEDStatusUpdate struct {
	Kind       string
	Power      *bool
	Brightness *int
	Color      *string
	PixelPin   *int
	RawJSON    string
}
