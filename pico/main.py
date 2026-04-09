# Directory: pico/
# Modified: 2026-04-08
# Description: MicroPython LED strip firmware. Connects to WiFi and MQTT, publishes status, handles commands, and prints detailed runtime logs.
# Uses: pico/device_config.py
# Used by: none (runs on Pico W as entry point)

import network
import time
import json
import ubinascii
import sys
from machine import Pin
import neopixel
from umqtt.simple import MQTTClient
import device_config as cfg

WIFI_SSID = cfg.WIFI_SSID
WIFI_PASSWORD = cfg.WIFI_PASSWORD
MQTT_BROKER = cfg.MQTT_BROKER
MQTT_PORT = cfg.MQTT_PORT
PIXEL_PIN = cfg.PIXEL_PIN
PIXEL_COUNT = cfg.PIXEL_COUNT
VERBOSE = getattr(cfg, "VERBOSE", True)
PROVISION_TAG = getattr(cfg, "PROVISION_TAG", "")


def log(msg):
    if VERBOSE:
        print("[pico] " + str(msg))

pixel = neopixel.NeoPixel(Pin(PIXEL_PIN), PIXEL_COUNT)
status = {
    "kind": "led_strip",
    "power": False,
    "brightness": 64,
    "color": "#00FF99",
    "pixelPin": PIXEL_PIN,
    "pixelCount": PIXEL_COUNT
}
if PROVISION_TAG:
    status["provisionTag"] = PROVISION_TAG


def connect_wifi():
    log("Starting WiFi connection")
    log("SSID: " + WIFI_SSID)
    wlan = network.WLAN(network.STA_IF)
    wlan.active(True)
    if not wlan.isconnected():
        log("WiFi not connected, attempting connect")
        wlan.connect(WIFI_SSID, WIFI_PASSWORD)
        wait = 0
        while not wlan.isconnected():
            wait += 1
            if wait % 5 == 0:
                log("Waiting for WiFi... " + str(wait) + "s")
            time.sleep(1)
    log("WiFi connected, ifconfig=" + str(wlan.ifconfig()))
    return wlan


def get_mac():
    mac = ubinascii.hexlify(network.WLAN(network.STA_IF).config('mac')).decode()
    formatted = ':'.join([mac[i:i + 2] for i in range(0, 12, 2)])
    log("Detected MAC: " + formatted)
    return formatted


def hex_to_rgb(h):
    h = h.lstrip('#')
    return tuple(int(h[i:i + 2], 16) for i in (0, 2, 4))


def apply_led():
    if not status["power"]:
        rgb = (0, 0, 0)
    else:
        base = hex_to_rgb(status["color"])
        factor = max(0, min(255, status["brightness"])) / 255
        rgb = (int(base[0] * factor), int(base[1] * factor), int(base[2] * factor))
    for i in range(PIXEL_COUNT):
        pixel[i] = rgb
    pixel.write()
    log("Applied LED state power=" + str(status["power"]) + " brightness=" + str(status["brightness"]) + " color=" + str(status["color"]))


def on_message(topic, msg):
    global status
    try:
        log("MQTT command received on " + topic.decode())
        log("Raw payload: " + msg.decode())
        cmd = json.loads(msg)
        if "power" in cmd:
            status["power"] = bool(cmd["power"])
        if "brightness" in cmd:
            status["brightness"] = int(cmd["brightness"])
        if "color" in cmd:
            status["color"] = str(cmd["color"])
        if "pixelPin" in cmd:
            status["pixelPin"] = int(cmd["pixelPin"])
            log("pixelPin command received: " + str(status["pixelPin"]) + " (hardware pin remains configured at boot)")
        apply_led()
    except Exception as e:
        print("[pico] cmd parse error", e)
        sys.print_exception(e)


def main():
    log("Booting Pico LED firmware")
    log("Config MQTT broker=" + MQTT_BROKER + ":" + str(MQTT_PORT) + " pixelPin=" + str(PIXEL_PIN) + " pixelCount=" + str(PIXEL_COUNT))
    connect_wifi()
    mac = get_mac()
    status_topic = b"devices/status/" + mac.encode()
    cmd_topic = b"devices/cmd/" + mac.encode()
    log("Status topic: " + status_topic.decode())
    log("Command topic: " + cmd_topic.decode())

    client = MQTTClient(client_id=mac, server=MQTT_BROKER, port=MQTT_PORT)
    client.set_callback(on_message)
    connected = False
    attempts = 0
    while not connected:
        attempts += 1
        try:
            log("Connecting MQTT attempt " + str(attempts))
            client.connect(clean_session=True)
            connected = True
        except Exception as e:
            print("[pico] mqtt connect error", e)
            time.sleep(2)
    client.subscribe(cmd_topic, qos=1)
    log("Subscribed to command topic (QoS 1)")

    apply_led()

    publish_count = 0
    while True:
        try:
            client.check_msg()
            payload = json.dumps(status)
            client.publish(status_topic, payload, qos=1)
            publish_count += 1
            log("Published status #" + str(publish_count) + " " + payload)
        except Exception as e:
            print("[pico] mqtt loop error", e)
            sys.print_exception(e)
        time.sleep(8)


main()
