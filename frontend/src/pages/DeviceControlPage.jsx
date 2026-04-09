// Directory: frontend/src/pages/
// Modified: 2026-04-08
// Description: Per-device editor and control page. Supports rename/room updates plus LED power, brightness, and color controls.
// Uses: frontend/src/api.js
// Used by: frontend/src/App.jsx

import { useEffect, useMemo, useState } from 'react';
import { Button, Card, ColorPicker, Input, Select, Slider, Space, Switch, Typography, message } from 'antd';
import { useParams } from 'react-router-dom';
import { api } from '../api';

export default function DeviceControlPage() {
  const { mac } = useParams();
  const [devices, setDevices] = useState([]);
  const [rooms, setRooms] = useState([]);
  const [power, setPower] = useState(false);
  const [brightness, setBrightness] = useState(0);
  const [colorHex, setColorHex] = useState('#000000');
  const [name, setName] = useState('');
  const [roomId, setRoomId] = useState(null);

  useEffect(() => {
    (async () => {
      try {
        const [all, roomList] = await Promise.all([api.listDevices(), api.listRooms()]);
        setDevices(all);
        setRooms(roomList);
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
    setName(device.name || '');
    setRoomId(device.roomId ?? null);
  }, [device]);

  async function send(body) {
    try {
      await api.sendCommand(mac, body);
      setDevices((prev) => prev.map((d) => (d.mac === mac ? { ...d, ledStrip: { ...d.ledStrip, ...body } } : d)));
      message.success('Applied');
    } catch (err) {
      message.error(err.message);
    }
  }

  async function saveDeviceDetails() {
    try {
      await api.updateDevice(mac, { name: name.trim(), roomId });
      const all = await api.listDevices();
      setDevices(all);
      message.success('Device details saved');
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
        <Card size="small" title="Device Details">
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Device name" />
            <Select
              value={roomId}
              options={rooms.map((r) => ({ value: r.id, label: r.name }))}
              placeholder="Select room"
              onChange={(v) => setRoomId(v)}
            />
            <Button type="primary" onClick={saveDeviceDetails}>Save details</Button>
          </Space>
        </Card>

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
        <Space direction="vertical" style={{ width: '100%' }}>
          <Typography.Text>Brightness</Typography.Text>
          <Slider
            min={0}
            max={255}
            value={brightness}
            onChange={(v) => setBrightness(Number(v ?? 0))}
            onAfterChange={(v) => send({ brightness: Number(v ?? 0) })}
          />
        </Space>
        <Space>
          <Typography.Text>Color</Typography.Text>
          <ColorPicker
            value={colorHex}
            onChange={(_, hex) => {
              setColorHex(hex);
            }}
            onChangeComplete={(v) => {
              const hex = v.toHexString();
              setColorHex(hex);
              send({ color: hex });
            }}
            showText
          />
        </Space>
      </Space>
    </Card>
  );
}
