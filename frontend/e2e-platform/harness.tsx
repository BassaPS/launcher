import { StrictMode, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { usePlatformInfo } from '../src/hooks/usePlatformInfo';

function Consumer({ id }: { id: number }) {
  const info = usePlatformInfo();
  return <section>
    <output data-testid={`consumer-${id}`}>{JSON.stringify(info)}</output>
    <button onClick={info.refreshStatus}>retry status</button>
    <button onClick={info.refreshRelease}>retry release</button>
  </section>;
}

function App() {
  const [count, setCount] = useState(2);
  const visibility = (hidden: boolean) => {
    Object.defineProperty(document, 'hidden', { configurable: true, value: hidden });
    document.dispatchEvent(new Event('visibilitychange'));
  };
  return <>
    <button onClick={() => setCount(0)}>none</button>
    <button onClick={() => setCount(1)}>one</button>
    <button onClick={() => setCount(2)}>two</button>
    <button onClick={() => visibility(true)}>hidden</button>
    <button onClick={() => visibility(false)}>visible</button>
    {Array.from({ length: count }, (_, id) => <Consumer key={id} id={id} />)}
  </>;
}

createRoot(document.getElementById('root')!).render(<StrictMode><App /></StrictMode>);
