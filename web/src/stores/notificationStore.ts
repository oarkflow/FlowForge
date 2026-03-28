import { createSignal, createMemo } from 'solid-js';

interface Notification {
  id: string;
  type: 'info' | 'success' | 'warning' | 'error';
  title: string;
  message?: string;
  timestamp: Date;
  read: boolean;
  link?: string;
}

const [notifications, setNotifications] = createSignal<Notification[]>([]);

const unreadCount = createMemo(() => notifications().filter((n) => !n.read).length);

const addNotification = (notification: Omit<Notification, 'id' | 'timestamp' | 'read'>) => {
  const newNotification: Notification = {
    ...notification,
    id: crypto.randomUUID(),
    timestamp: new Date(),
    read: false,
  };
  setNotifications((prev) => [newNotification, ...prev].slice(0, 100)); // Keep last 100
};

const markRead = (id: string) => {
  setNotifications((prev) =>
    prev.map((n) => (n.id === id ? { ...n, read: true } : n))
  );
};

const markAllRead = () => {
  setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
};

const clearAll = () => {
  setNotifications([]);
};

const removeNotification = (id: string) => {
  setNotifications((prev) => prev.filter((n) => n.id !== id));
};

export const notificationStore = {
  notifications,
  unreadCount,
  addNotification,
  markRead,
  markAllRead,
  clearAll,
  removeNotification,
};

export type { Notification };
