// Directory: frontend/src/pages/
// Modified: 2026-04-08
// Description: Per-device editor and control page. Supports rename/room updates plus LED power, brightness, and color controls.
// Uses: frontend/src/api.js
// Used by: frontend/src/App.jsx

import { useEffect, useMemo, useState } from 'react';
import { Button, Card, Input, Select, Slider, Space, Switch, Typography, message } from 'antd';
import { useParams } from 'react-router-dom';
import { api } from '../api';

const OFFLINE_AFTER_MS = 10_000;
const POLL_MS = 2000;
const DISCRETE_COLORS = [
  '#FF3B30', '#FF9500', '#FFCC00', '#34C759', '#00C7BE', '#32ADE6', '#007AFF', '#5856D6',
  '#AF52DE', '#FF2D55', '#FFFFFF', '#C7C7CC', '#8E8E93', '#636366', '#3A3A3C', '#000000'
];

function isOnline(lastSeen) {
  if (!lastSeen) return false;
  const ts = Date.parse(lastSeen);
  if (Number.isNaN(ts)) return false;
  const ageMs = Math.max(0, Date.now() - ts);
  return ageMs <= OFFLINE_AFTER_MS;
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

  async function loadDetails(silent = false) {
    try {
      const [all, roomList] = await Promise.all([api.listDevices(), api.listRooms()]);
      setDevices(all);
      setRooms(roomList);
    } catch (err) {
      if (!silent) {
        message.error(err.message);
      }
    }
  }

  useEffect(() => {
    loadDetails();
    const t = setInterval(() => loadDetails(true), POLL_MS);
    return () => clearInterval(t);
  }, [mac]);

  const device = useMemo(() => devices.find((d) => d.mac === mac), [devices, mac]);
  const connected = isOnline(device?.lastSeen);

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
              <Typography.Text><Typography.Text strong>Status:</Typography.Text> {connected ? 'Connected' : 'Disconnected'}</Typography.Text>
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
        </Space>
        <Space size={8} wrap>
          {DISCRETE_COLORS.map((hex) => (
            <Button
              key={hex}
              shape="circle"
              size="small"
              title={hex}
              onClick={() => {
                setColorHex(hex);
                send({ color: hex });
              }}
              style={{
                backgroundColor: hex,
                borderColor: colorHex.toLowerCase() === hex.toLowerCase() ? '#1f7a56' : '#d9d9d9',
                borderWidth: colorHex.toLowerCase() === hex.toLowerCase() ? 2 : 1
              }}
            />
          ))}
        </Space>
        <Typography.Text type="secondary">Selected color: {colorHex}</Typography.Text>
      </Space>
    </Card>
  );
}
