// Directory: frontend/src/pages/
// Modified: 2026-04-08
// Description: Dedicated Pico provisioning page. Detects USB mode, accepts pixel pin, and shows step-by-step progress.
// Uses: frontend/src/api.js
// Used by: frontend/src/App.jsx

import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Card, Descriptions, InputNumber, Modal, Space, Steps, Tag, Typography, message } from 'antd';
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
  const [provState, setProvState] = useState({
    running: false,
    stage: 'idle',
    detail: 'Waiting for configure request',
    lastResult: 'none',
    attempt: 0,
    startedAt: '',
    finishedAt: ''
  });
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

  async function resetProvisionState() {
    try {
      await api.picoProvisionReset();
      const state = await api.picoProvisionState();
      setProvState(state);
      message.success('Provisioning status cleared');
    } catch (err) {
      message.error(err.message);
    }
  }

  const currentStep = STAGE_INDEX[provState.stage] ?? 0;
  const overallStatus = provState.stage === 'error' ? 'error' : provState.stage === 'done' ? 'finish' : 'process';

  const runStatusTag = provState.running
    ? { color: 'processing', text: 'Running' }
    : provState.stage === 'error'
      ? { color: 'error', text: 'Failed' }
      : provState.stage === 'done'
        ? { color: 'success', text: 'Succeeded' }
        : { color: 'default', text: 'Idle' };

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
            <Descriptions size="small" column={1} bordered>
              <Descriptions.Item label="Run status">
                <Tag color={runStatusTag.color}>{runStatusTag.text}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Attempt">
                {provState.attempt > 0 ? `#${provState.attempt}` : 'None yet'}
              </Descriptions.Item>
              <Descriptions.Item label="Started at">
                {provState.startedAt || 'Not started'}
              </Descriptions.Item>
              <Descriptions.Item label="Finished at">
                {provState.finishedAt || 'Not finished'}
              </Descriptions.Item>
              <Descriptions.Item label="Last update">
                {provState.updatedAt || 'Unknown'}
              </Descriptions.Item>
            </Descriptions>

            <Typography.Text strong>{provState.detail || 'Waiting for configure request'}</Typography.Text>
            {provState.error ? (
              <Alert
                type="error"
                showIcon
                message={`Attempt #${provState.attempt || '?'} failed`}
                description={provState.error}
                action={
                  <Button size="small" onClick={resetProvisionState} disabled={provState.running}>
                    Clear
                  </Button>
                }
              />
            ) : null}
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
          Flow: plug in Pico, click Configure, watch the active attempt status and timestamps, then disconnect after Succeeded.
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
