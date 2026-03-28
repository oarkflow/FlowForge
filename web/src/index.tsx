/* @refresh reload */
import { render } from 'solid-js/web';
import './index.css';
import App from './App';

const root = document.getElementById('root');

if (!root) {
  throw new Error('Root element not found. Add <div id="root"></div> to your index.html.');
}

render(() => <App />, root);
