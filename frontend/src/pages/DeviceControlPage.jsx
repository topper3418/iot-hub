import { useEffect, useMemo, useState } from 'react';
import { Button, Card, ColorPicker, InputNumber, Space, Switch, Typography, message } from 'antd';
import { useParams } from 'react-router-dom';
import { api } from '../api';

export default function DeviceControlPage() {
  const { mac } = useParams();
  const [devices, setDevices] = useState([]);
  const [power, setPower] = useState(false);
  const [brightness, setBrightness] = useState(0);
  const [colorHex, setColorHex] = useState('#000000');
  const [pixelPin, setPixelPin] = useState(0);

  useEffect(() => {
    (async () => {
      try {
        const all = await api.listDevices();
        setDevices(all);
      } catch (err) {
        message.error(err.message);
      }
    })();
  }, [mac]);

  const device = useMemo(() => devices.find((d) => d.mac === mac), [devices, mac]);

  useEffect(() => {
    if (!device) return;
    setPower(Boolean(device.ledStrip?.power));
    setBrightness(device.ledStrip?.brightness ?? 0);
    setColorHex(device.ledStrip?.color || '#000000');
    setPixelPin(device.ledStrip?.pixelPin ?? 0);
  }, [device]);

  async function send(body) {
    try {
      await api.sendCommand(mac, body);
      message.success('Command sent');
    } catch (err) {
      message.error(err.message);
    }
  }

  if (!device) {
    return <Card className="control-card">Device not found.</Card>;
  }

  return (
    <Card className="control-card" title={`Device Control: ${device.name}`}>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Space>
          <Typography.Text>Power</Typography.Text>
          <Switch
            checked={power}
            onChange={(v) => {
              setPower(v);
              send({ power: v });
            }}
          />
        </Space>
        <Space>
          <Typography.Text>Brightness</Typography.Text>
          <InputNumber
            min={0}
            max={255}
            value={brightness}
            onChange={(v) => setBrightness(v ?? 0)}
          />
          <Button type="primary" onClick={() => send({ brightness })}>Apply</Button>
        </Space>
        <Space>
          <Typography.Text>Color</Typography.Text>
          <ColorPicker value={colorHex} onChange={(_, hex) => setColorHex(hex)} showText />
          <Button type="primary" onClick={() => send({ color: colorHex })}>Apply</Button>
        </Space>
        <Space>
          <Typography.Text>Pixel Pin</Typography.Text>
          <InputNumber min={0} max={39} value={pixelPin} onChange={(v) => setPixelPin(v ?? 0)} />
          <Button type="primary" onClick={() => send({ pixelPin })}>Apply</Button>
        </Space>
      </Space>
    </Card>
  );
}
