const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

async function request(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
    ...options
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `Request failed: ${res.status}`);
  }
  if (res.status === 204) return null;
  return res.json();
}

export const api = {
  listDevices: () => request('/api/devices'),
  updateDevice: (mac, body) => request(`/api/devices/${mac}/update`, { method: 'PUT', body: JSON.stringify(body) }),
  sendCommand: (mac, body) => request(`/api/devices/${mac}/command`, { method: 'POST', body: JSON.stringify(body) }),
  listRooms: () => request('/api/rooms'),
  createRoom: (name) => request('/api/rooms', { method: 'POST', body: JSON.stringify({ name }) })
};
