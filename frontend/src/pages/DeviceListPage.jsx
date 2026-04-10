// Directory: frontend/src/pages/
// Modified: 2026-04-08
// Description: Devices overview page using clickable cards with quick on/off controls.
// Uses: frontend/src/api.js
// Used by: frontend/src/App.jsx

import { useEffect, useState } from 'react';
import { Card, Empty, List, Space, Switch, Tag, Typography, message } from 'antd';
import { useNavigate } from 'react-router-dom';
import { api } from '../api';

const OFFLINE_AFTER_MS = 10_000;

function msSince(value, nowMs) {
  if (!value) return Number.POSITIVE_INFINITY;
  const ts = Date.parse(value);
  if (Number.isNaN(ts)) return Number.POSITIVE_INFINITY;
  return Math.max(0, nowMs - ts);
}

export default function DeviceListPage() {
  const navigate = useNavigate();
  const [devices, setDevices] = useState([]);
  const [loading, setLoading] = useState(true);
  const [nowMs, setNowMs] = useState(Date.now());

  async function load() {
    setLoading(true);
    try {
      const d = await api.listDevices();
      setDevices(d);
    } catch (err) {
      message.error(err.message);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    const t = setInterval(load, 5000);
    return () => clearInterval(t);
  }, []);

  useEffect(() => {
    const t = setInterval(() => setNowMs(Date.now()), 1000);
    return () => clearInterval(t);
  }, []);

  async function sendCommand(mac, body) {
    try {
      await api.sendCommand(mac, body);
      setDevices((prev) => prev.map((d) => (d.mac === mac ? { ...d, ledStrip: { ...d.ledStrip, ...body } } : d)));
    } catch (err) {
      message.error(err.message);
    }
  }

  return (
    <Card className="control-card" title="Devices">
      <List
        loading={loading}
        locale={{ emptyText: <Empty description="No devices seen yet" /> }}
        grid={{ gutter: 16, xs: 1, sm: 1, md: 2, lg: 3 }}
        dataSource={devices}
        renderItem={(device) => {
          const ageMs = msSince(device.lastSeen, nowMs);
          const isOnline = ageMs <= OFFLINE_AFTER_MS;
          const seenLabel = Number.isFinite(ageMs) ? `${Math.floor(ageMs / 1000)}s ago` : 'never';
          return (
            <List.Item>
              <Card
                hoverable
                onClick={() => navigate(`/device/${device.mac}`)}
                title={device.name}
                extra={<Tag>{device.roomName || 'Unassigned'}</Tag>}
              >
                <Space direction="vertical" size={8} style={{ width: '100%' }}>
                  <Space>
                    <Tag color={isOnline ? 'green' : 'red'}>{isOnline ? 'Connected' : 'Disconnected'}</Tag>
                    <Typography.Text type="secondary">Last seen {seenLabel}</Typography.Text>
                  </Space>
                  <Space align="center" onClick={(e) => e.stopPropagation()}>
                    <Typography.Text>Power</Typography.Text>
                    <Switch
                      checked={Boolean(device.ledStrip?.power)}
                      onChange={(v) => sendCommand(device.mac, { power: v })}
                    />
                  </Space>
                </Space>
              </Card>
            </List.Item>
          );
        }}
      />
    </Card>
  );
}
