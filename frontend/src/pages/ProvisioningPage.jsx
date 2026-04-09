// Directory: frontend/src/pages/
// Modified: 2026-04-08
// Description: Dedicated Pico provisioning page. Detects USB mode, accepts pixel pin, and shows step-by-step progress.
// Uses: frontend/src/api.js
// Used by: frontend/src/App.jsx

import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Card, InputNumber, Modal, Space, Steps, Typography, message } from 'antd';
import { api } from '../api';

const STAGE_INDEX = {
  idle: 0,
  queued: 0,
  validating: 1,
  network: 1,
  flashing: 2,
  waiting_serial: 3,
  serial: 3,
  config: 4,
  upload_main: 5,
  upload_config: 6,
  reset: 7,
  done: 8,
  error: 8
};

export default function ProvisioningPage() {
  const [picoStatus, setPicoStatus] = useState({ state: 'none', connected: false });
  const [provState, setProvState] = useState({ running: false, stage: 'idle', detail: 'Waiting for configure request', lastResult: 'none' });
  const [showModal, setShowModal] = useState(false);
  const [pixelPin, setPixelPin] = useState(16);
  const [starting, setStarting] = useState(false);

  useEffect(() => {
    let mounted = true;

    async function poll() {
      try {
        const [status, state] = await Promise.all([api.picoStatus(), api.picoProvisionState()]);
        if (!mounted) return;
        setPicoStatus(status);
        setProvState(state);
      } catch (err) {
        if (mounted) {
          message.error(err.message);
        }
      }
    }

    poll();
    const t = setInterval(poll, 1500);
    return () => {
      mounted = false;
      clearInterval(t);
    };
  }, []);

  const canConfigure = picoStatus.state === 'bootsel' || picoStatus.state === 'micropython';

  const statusAlert = useMemo(() => {
    if (picoStatus.state === 'bootsel') {
      return {
        type: 'success',
        message: 'Pico in BOOTSEL mode',
        description: 'Ready for first-time flash and file upload.'
      };
    }
    if (picoStatus.state === 'micropython') {
      return {
        type: 'info',
        message: 'Pico in MicroPython mode',
        description: picoStatus.serialPort ? `Detected on ${picoStatus.serialPort}` : 'Ready to receive files.'
      };
    }
    return {
      type: 'warning',
      message: 'No Pico connected',
      description: 'Hold BOOTSEL while plugging in, then click Configure.'
    };
  }, [picoStatus]);

  async function startProvision() {
    setStarting(true);
    try {
      await api.picoProvision({ pixelPin });
      message.success('Provisioning started');
      setShowModal(false);
    } catch (err) {
      message.error(err.message);
    } finally {
      setStarting(false);
    }
  }

  const currentStep = STAGE_INDEX[provState.stage] ?? 0;
  const overallStatus = provState.stage === 'error' ? 'error' : provState.stage === 'done' ? 'finish' : 'process';

  return (
    <Card className="control-card" title="Pico Provisioning">
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Alert
          showIcon
          {...statusAlert}
          action={
            <Button type="primary" disabled={!canConfigure || provState.running} onClick={() => setShowModal(true)}>
              Configure
            </Button>
          }
        />

        <Card size="small" title="Provisioning Progress">
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Typography.Text strong>{provState.detail || 'Waiting for configure request'}</Typography.Text>
            {provState.error ? <Typography.Text type="danger">{provState.error}</Typography.Text> : null}
            <Steps
              current={currentStep}
              status={overallStatus}
              size="small"
              items={[
                { title: 'Queued' },
                { title: 'Validate' },
                { title: 'Flash UF2' },
                { title: 'Serial Ready' },
                { title: 'Config' },
                { title: 'Upload main.py' },
                { title: 'Upload device_config.py' },
                { title: 'Reset Pico' },
                { title: 'Done' }
              ]}
            />
          </Space>
        </Card>

        <Typography.Text type="secondary">
          Flow: plug in Pico, click Configure, wait for Done, then disconnect and power it normally.
        </Typography.Text>
      </Space>

      <Modal
        title="Configure Pico"
        open={showModal}
        okText="Configure"
        okButtonProps={{ loading: starting }}
        onOk={startProvision}
        onCancel={() => {
          if (!starting) {
            setShowModal(false);
          }
        }}
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Typography.Text>Set LED pixel pin (default GP16).</Typography.Text>
          <InputNumber min={0} max={28} value={pixelPin} onChange={(v) => setPixelPin(Number(v ?? 16))} />
          <Typography.Text type="secondary">Cancel closes this dialog without changing anything.</Typography.Text>
        </Space>
      </Modal>
    </Card>
  );
}
