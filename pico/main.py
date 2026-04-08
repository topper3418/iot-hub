import network
import time
import json
import ubinascii
from machine import Pin
import neopixel
from umqtt.simple import MQTTClient
from device_config import WIFI_SSID, WIFI_PASSWORD, MQTT_BROKER, MQTT_PORT, PIXEL_PIN, PIXEL_COUNT

pixel = neopixel.NeoPixel(Pin(PIXEL_PIN), PIXEL_COUNT)
status = {
    "kind": "led_strip",
    "power": False,
    "brightness": 64,
    "color": "#00FF99",
    "pixelPin": PIXEL_PIN,
    "pixelCount": PIXEL_COUNT
}


def connect_wifi():
    wlan = network.WLAN(network.STA_IF)
    wlan.active(True)
    if not wlan.isconnected():
        wlan.connect(WIFI_SSID, WIFI_PASSWORD)
        while not wlan.isconnected():
            time.sleep(1)
    return wlan


def get_mac():
    mac = ubinascii.hexlify(network.WLAN(network.STA_IF).config('mac')).decode()
    return ':'.join([mac[i:i + 2] for i in range(0, 12, 2)])


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


def on_message(topic, msg):
    global status
    try:
        cmd = json.loads(msg)
        if "power" in cmd:
            status["power"] = bool(cmd["power"])
        if "brightness" in cmd:
            status["brightness"] = int(cmd["brightness"])
        if "color" in cmd:
            status["color"] = str(cmd["color"])
        if "pixelPin" in cmd:
            status["pixelPin"] = int(cmd["pixelPin"])
        apply_led()
    except Exception as e:
        print("cmd parse error", e)


def main():
    connect_wifi()
    mac = get_mac()
    status_topic = b"devices/status/" + mac.encode()
    cmd_topic = b"devices/cmd/" + mac.encode()

    client = MQTTClient(client_id=mac, server=MQTT_BROKER, port=MQTT_PORT)
    client.set_callback(on_message)
    client.connect(clean_session=True)
    client.subscribe(cmd_topic, qos=1)

    apply_led()

    while True:
        client.check_msg()
        client.publish(status_topic, json.dumps(status), qos=1)
        time.sleep(8)


main()
