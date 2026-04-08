import { useEffect, useMemo, useState } from 'react';
import { Button, Card, Input, Select, Space, Switch, Table, Typography, message } from 'antd';
import { Link } from 'react-router-dom';
import { api } from '../api';

export default function DeviceListPage() {
  const [devices, setDevices] = useState([]);
  const [rooms, setRooms] = useState([]);
  const [loading, setLoading] = useState(true);

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
