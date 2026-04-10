// Directory: frontend/src/pages/
// Modified: 2026-04-08
// Description: Per-device editor and control page. Supports rename/room updates plus LED power, brightness, and color controls.
// Uses: frontend/src/api.js
// Used by: frontend/src/App.jsx

import { useEffect, useMemo, useState } from 'react';
import { Button, Card, ColorPicker, Input, Select, Slider, Space, Switch, Typography, message } from 'antd';
import { useParams } from 'react-router-dom';
import { api } from '../api';

const OFFLINE_AFTER_MS = 10_000;

function connectionState(lastSeen, nowMs) {
  if (!lastSeen) return { online: false, text: 'never' };
  const ts = Date.parse(lastSeen);
  if (Number.isNaN(ts)) return { online: false, text: 'unknown' };
  const ageMs = Math.max(0, nowMs - ts);
  return {
    online: ageMs <= OFFLINE_AFTER_MS,
    text: `${Math.floor(ageMs / 1000)}s ago`
  };
}

export default function DeviceControlPage() {
  const { mac } = useParams();
  const [devices, setDevices] = useState([]);
  const [rooms, setRooms] = useState([]);
  const [power, setPower] = useState(false);
  const [brightness, setBrightness] = useState(0);
  const [colorHex, setColorHex] = useState('#000000');
  const [name, setName] = useState('');
  const [roomId, setRoomId] = useState(null);
  const [editingDetails, setEditingDetails] = useState(false);
  const [nowMs, setNowMs] = useState(Date.now());

  async function loadDetails() {
    try {
      const [all, roomList] = await Promise.all([api.listDevices(), api.listRooms()]);
      setDevices(all);
      setRooms(roomList);
    } catch (err) {
      message.error(err.message);
    }
  }

  useEffect(() => {
    loadDetails();
    const t = setInterval(loadDetails, 5000);
    return () => clearInterval(t);
  }, [mac]);

  useEffect(() => {
    const t = setInterval(() => setNowMs(Date.now()), 1000);
    return () => clearInterval(t);
  }, []);

  const device = useMemo(() => devices.find((d) => d.mac === mac), [devices, mac]);
  const conn = useMemo(() => connectionState(device?.lastSeen, nowMs), [device?.lastSeen, nowMs]);

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
      await loadDetails();
      setEditingDetails(false);
      message.success('Device details saved');
    } catch (err) {
      message.error(err.message);
    }
  }

  function cancelEditDetails() {
    if (!device) return;
    setName(device.name || '');
    setRoomId(device.roomId ?? null);
    setEditingDetails(false);
  }

  if (!device) {
    return <Card className="control-card">Device not found.</Card>;
  }

  return (
    <Card className="control-card" title={`Device Control: ${device.name}`}>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Card size="small" title="Device Details">
          {!editingDetails ? (
            <Space direction="vertical" size={10} style={{ width: '100%' }}>
              <Typography.Text><Typography.Text strong>Name:</Typography.Text> {device.name || 'Unnamed'}</Typography.Text>
              <Typography.Text><Typography.Text strong>Room:</Typography.Text> {device.roomName || 'Unassigned'}</Typography.Text>
              <Typography.Text><Typography.Text strong>Status:</Typography.Text> {conn.online ? 'Connected' : 'Disconnected'} (last seen {conn.text})</Typography.Text>
              <Button onClick={() => setEditingDetails(true)}>Edit</Button>
            </Space>
          ) : (
            <Space direction="vertical" size={12} style={{ width: '100%' }}>
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Device name" />
              <Select
                value={roomId}
                options={rooms.map((r) => ({ value: r.id, label: r.name }))}
                placeholder="Select room"
                onChange={(v) => setRoomId(v)}
              />
              <Space>
                <Button type="primary" onClick={saveDeviceDetails}>Save details</Button>
                <Button onClick={cancelEditDetails}>Cancel</Button>
              </Space>
            </Space>
          )}
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
