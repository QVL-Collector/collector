import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';
import { createRoot } from 'react-dom/client';
import { startApplication } from './app/startApplication';

const rootElement = document.getElementById('root');

if (rootElement === null) {
  throw new Error('Root element #root is missing');
}

void startApplication(createRoot(rootElement));
