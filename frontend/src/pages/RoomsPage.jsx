// Directory: frontend/src/pages/
// Modified: 2026-04-08
// Description: Rooms management page. Lists existing rooms and allows creating new ones.
// Uses: frontend/src/api.js
// Used by: frontend/src/App.jsx

import { useEffect, useState } from 'react';
import { Button, Card, Input, List, Space, Typography, message } from 'antd';
import { api } from '../api';

export default function RoomsPage() {
  const [rooms, setRooms] = useState([]);
  const [name, setName] = useState('');

  async function loadRooms() {
    try {
      setRooms(await api.listRooms());
    } catch (err) {
      message.error(err.message);
    }
  }

  useEffect(() => {
    loadRooms();
  }, []);

  async function addRoom() {
    if (!name.trim()) return;
    try {
      await api.createRoom(name.trim());
      setName('');
      await loadRooms();
      message.success('Room saved');
    } catch (err) {
      message.error(err.message);
    }
  }

  return (
    <Card className="control-card" title="Rooms">
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        <Space.Compact style={{ width: '100%' }}>
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Add room name"
            onPressEnter={addRoom}
          />
          <Button type="primary" onClick={addRoom}>Add</Button>
        </Space.Compact>
        <List
          bordered
          dataSource={rooms}
          renderItem={(room) => (
            <List.Item>
              <Typography.Text>{room.name}</Typography.Text>
            </List.Item>
          )}
        />
      </Space>
    </Card>
  );
}
