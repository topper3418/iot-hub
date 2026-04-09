// Directory: frontend/src/pages/
// Modified: 2026-04-08
// Description: Device list page with inline LED controls, device rename, room assignment, and Pico USB status.
// Uses: frontend/src/api.js
// Used by: frontend/src/App.jsx

import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Card, Input, Select, Space, Switch, Table, Typography, message } from 'antd';
import { Link } from 'react-router-dom';
import { api } from '../api';

export default function DeviceListPage() {
  const [devices, setDevices] = useState([]);
  const [rooms, setRooms] = useState([]);
  const [loading, setLoading] = useState(true);
  const [picoStatus, setPicoStatus] = useState({ state: 'none', connected: false });

  async function load() {
    setLoading(true);
    try {
      const [d, r] = await Promise.all([api.listDevices(), api.listRooms()]);
      setDevices(d);
      setRooms(r);
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
    let mounted = true;

    async function loadPicoStatus() {
      try {
        const status = await api.picoStatus();
        if (mounted) {
          setPicoStatus(status);
        }
      } catch {
        if (mounted) {
          setPicoStatus({ state: 'none', connected: false });
        }
      }
    }

    loadPicoStatus();
    const t = setInterval(loadPicoStatus, 2000);
    return () => {
      mounted = false;
      clearInterval(t);
    };
  }, []);

  function picoAlert() {
    if (picoStatus.state === 'bootsel') {
      return {
        type: 'success',
        message: 'Pico in BOOTSEL mode detected',
        description: 'Ready for first-time flash. Next step is provisioning upload wiring.'
      };
    }
    if (picoStatus.state === 'micropython') {
      return {
        type: 'info',
        message: 'Pico detected in MicroPython mode',
        description: picoStatus.serialPort ? `Serial: ${picoStatus.serialPort}` : 'Serial interface available'
      };
    }
    return {
      type: 'warning',
      message: 'No Pico detected',
      description: 'Hold BOOTSEL while plugging in to enter flash mode.'
    };
  }

  const roomOptions = useMemo(
    () => rooms.map((r) => ({ value: r.id, label: r.name })),
    [rooms]
  );

  async function patchDevice(mac, body) {
    try {
      await api.updateDevice(mac, body);
      await load();
      message.success('Device updated');
    } catch (err) {
      message.error(err.message);
    }
  }

  async function sendCommand(mac, body) {
    try {
      await api.sendCommand(mac, body);
      message.success('Command sent');
    } catch (err) {
      message.error(err.message);
    }
  }

  const columns = [
    {
      title: 'Name',
      render: (_, row) => (
        <Space direction="vertical" size={4}>
          <Typography.Text strong>{row.name}</Typography.Text>
          <Typography.Text type="secondary">{row.mac}</Typography.Text>
        </Space>
      )
    },
    {
      title: 'Room',
      render: (_, row) => (
        <Select
          value={row.roomId}
          options={roomOptions}
          style={{ width: 180 }}
          onChange={(roomId) => patchDevice(row.mac, { roomId })}
        />
      )
    },
    {
      title: 'Rename',
      render: (_, row) => (
        <Input.Search
          placeholder="New device name"
          enterButton="Save"
          onSearch={(v) => v.trim() && patchDevice(row.mac, { name: v.trim() })}
        />
      )
    },
    {
      title: 'Power',
      render: (_, row) => (
        <Switch checked={Boolean(row.ledStrip?.power)} onChange={(v) => sendCommand(row.mac, { power: v })} />
      )
    },
    {
      title: 'Brightness',
      render: (_, row) => (
        <Input.Search
          placeholder="0-255"
          enterButton="Set"
          onSearch={(v) => {
            const n = Number(v);
            if (!Number.isInteger(n) || n < 0 || n > 255) {
              message.error('Brightness must be 0-255');
              return;
            }
            sendCommand(row.mac, { brightness: n });
          }}
        />
      )
    },
    {
      title: 'Color',
      render: (_, row) => (
        <Space>
          <Input
            defaultValue={row.ledStrip?.color || '#000000'}
            style={{ width: 120 }}
            onPressEnter={(e) => sendCommand(row.mac, { color: e.currentTarget.value })}
          />
          <Button type="link">
            <Link to={`/device/${row.mac}`}>Open</Link>
          </Button>
        </Space>
      )
    }
  ];

  return (
    <Card className="control-card" title="Devices">
      <Alert style={{ marginBottom: 16 }} showIcon {...picoAlert()} />
      <Table
        rowKey="mac"
        loading={loading}
        dataSource={devices}
        columns={columns}
        pagination={false}
      />
    </Card>
  );
}
