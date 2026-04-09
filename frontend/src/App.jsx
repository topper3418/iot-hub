// Directory: frontend/src/
// Modified: 2026-04-08
// Description: Root application component. Defines layout, navigation, and top-level route structure.
// Uses: frontend/src/pages/DeviceListPage.jsx, frontend/src/pages/RoomsPage.jsx, frontend/src/pages/DeviceControlPage.jsx
// Used by: frontend/src/main.jsx

import { Layout, Menu, Typography } from 'antd';
import { BulbOutlined, HomeOutlined, PartitionOutlined } from '@ant-design/icons';
import { Link, Route, Routes, useLocation } from 'react-router-dom';
import DeviceListPage from './pages/DeviceListPage';
import RoomsPage from './pages/RoomsPage';
import DeviceControlPage from './pages/DeviceControlPage';

const { Header, Content } = Layout;

export default function App() {
  const location = useLocation();
  const selected = location.pathname.startsWith('/rooms')
    ? '/rooms'
    : location.pathname.startsWith('/device/')
      ? '/devices'
      : '/devices';

  return (
    <Layout className="app-shell">
      <Header style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
        <Typography.Text className="brand">
          <BulbOutlined /> HOME LED CONTROL
        </Typography.Text>
        <Menu
          mode="horizontal"
          theme="dark"
          selectedKeys={[selected]}
          items={[
            { key: '/devices', icon: <HomeOutlined />, label: <Link to="/">Devices</Link> },
            { key: '/rooms', icon: <PartitionOutlined />, label: <Link to="/rooms">Rooms</Link> }
          ]}
        />
      </Header>
      <Content className="content-wrap">
        <Routes>
          <Route path="/" element={<DeviceListPage />} />
          <Route path="/rooms" element={<RoomsPage />} />
          <Route path="/device/:mac" element={<DeviceControlPage />} />
        </Routes>
      </Content>
    </Layout>
  );
}
