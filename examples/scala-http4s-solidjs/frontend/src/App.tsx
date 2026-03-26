import { createSignal, onMount, For } from 'solid-js';

interface Todo {
  id: number;
  title: string;
  completed: boolean;
}

export default function App() {
  const [todos, setTodos] = createSignal<Todo[]>([]);
  const [title, setTitle] = createSignal('');

  const fetchTodos = async () => {
    const res = await fetch('/api/todos');
    setTodos(await res.json());
  };

  onMount(fetchTodos);

  const addTodo = async () => {
    if (!title().trim()) return;
    await fetch('/api/todos', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title: title() }),
    });
    setTitle('');
    await fetchTodos();
  };

  const toggle = async (id: number) => {
    await fetch(`/api/todos/${id}/toggle`, { method: 'PUT' });
    await fetchTodos();
  };

  const remove = async (id: number) => {
    await fetch(`/api/todos/${id}`, { method: 'DELETE' });
    await fetchTodos();
  };

  return (
    <div style={{ 'max-width': '500px', margin: '2rem auto', 'font-family': 'sans-serif' }}>
      <h1>Todo App</h1>
      <div style={{ display: 'flex', gap: '8px', 'margin-bottom': '1rem' }}>
        <input
          value={title()}
          onInput={(e) => setTitle(e.currentTarget.value)}
          onKeyDown={(e) => e.key === 'Enter' && addTodo()}
          placeholder="What needs to be done?"
          style={{ flex: 1, padding: '8px' }}
        />
        <button onClick={addTodo}>Add</button>
      </div>
      <ul style={{ 'list-style': 'none', padding: 0 }}>
        <For each={todos()}>
          {(todo) => (
            <li style={{ display: 'flex', 'align-items': 'center', gap: '8px', padding: '4px 0' }}>
              <input type="checkbox" checked={todo.completed} onChange={() => toggle(todo.id)} />
              <span style={{ flex: 1, 'text-decoration': todo.completed ? 'line-through' : 'none' }}>
                {todo.title}
              </span>
              <button onClick={() => remove(todo.id)}>x</button>
            </li>
          )}
        </For>
      </ul>
    </div>
  );
}
