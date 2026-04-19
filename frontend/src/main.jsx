// Directory: frontend/src/
// Modified: 2026-04-08
// Description: React entry point. Mounts the app into the DOM with BrowserRouter and StrictMode.
// Uses: frontend/src/App.jsx, frontend/src/styles.css
// Used by: frontend/index.html

import React from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import 'antd/dist/reset.css';
import './styles.css';
import App from './App';

createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <BrowserRouter basename={import.meta.env.BASE_URL}>
      <App />
    </BrowserRouter>
  </React.StrictMode>
);
