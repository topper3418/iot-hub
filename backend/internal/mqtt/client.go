// Directory: backend/internal/mqtt/
// Modified: 2026-04-08
// Description: MQTT broker connection, status topic subscription, and command topic publish.
// Uses: none (no internal package imports)
// Used by: backend/internal/app/server.go

package mqtt

import (
	"fmt"
	"log"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	client paho.Client
}

type StatusHandler func(mac string, payload []byte)

func NewClient(brokerURL, clientID string, handler StatusHandler) (*Client, error) {
	opts := paho.NewClientOptions()
	opts.AddBroker(brokerURL)
	opts.SetClientID(clientID)
	opts.SetCleanSession(true)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(2 * time.Second)
	opts.SetOnConnectHandler(func(c paho.Client) {
		token := c.Subscribe("devices/status/#", 1, func(_ paho.Client, msg paho.Message) {
			topic := msg.Topic()
			parts := strings.Split(topic, "/")
			if len(parts) != 3 {
				return
			}
			mac := strings.ToLower(parts[2])
			handler(mac, msg.Payload())
		})
		token.Wait()
		if token.Error() != nil {
			log.Printf("mqtt subscribe failed: %v", token.Error())
		}
	})

	client := paho.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if token.Error() != nil {
		return nil, fmt.Errorf("mqtt connect: %w", token.Error())
	}

	return &Client{client: client}, nil
}

func (c *Client) PublishCommand(mac string, payload []byte) error {
	topic := fmt.Sprintf("devices/cmd/%s", strings.ToLower(mac))
	token := c.client.Publish(topic, 1, false, payload)
	token.Wait()
	return token.Error()
}

func (c *Client) Close() {
	if c.client != nil && c.client.IsConnected() {
		c.client.Disconnect(250)
	}
}
