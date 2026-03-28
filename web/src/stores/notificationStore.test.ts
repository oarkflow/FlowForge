import { describe, it, expect } from 'vitest';
import { createRoot } from 'solid-js';
import { notificationStore } from './notificationStore';

describe('notificationStore', () => {
  it('starts with empty notifications', () => {
    createRoot(() => {
      expect(notificationStore.notifications()).toEqual([]);
    });
  });

  it('unreadCount starts at 0', () => {
    createRoot(() => {
      expect(notificationStore.unreadCount()).toBe(0);
    });
  });

  it('addNotification adds a notification', () => {
    createRoot(() => {
      notificationStore.addNotification({
        type: 'info',
        title: 'Test Notification',
        message: 'This is a test',
      });
      const notifications = notificationStore.notifications();
      expect(notifications.length).toBeGreaterThanOrEqual(1);
      const last = notifications[0];
      expect(last.title).toBe('Test Notification');
      expect(last.type).toBe('info');
      expect(last.read).toBe(false);
    });
  });

  it('unreadCount increments when adding unread notification', () => {
    createRoot(() => {
      // Clear first
      notificationStore.clearAll();
      notificationStore.addNotification({ type: 'success', title: 'N1' });
      notificationStore.addNotification({ type: 'error', title: 'N2' });
      expect(notificationStore.unreadCount()).toBe(2);
    });
  });

  it('markRead marks a notification as read', () => {
    createRoot(() => {
      notificationStore.clearAll();
      notificationStore.addNotification({ type: 'info', title: 'Read Me' });
      const n = notificationStore.notifications()[0];
      expect(n.read).toBe(false);
      notificationStore.markRead(n.id);
      const updated = notificationStore.notifications().find(x => x.id === n.id);
      expect(updated?.read).toBe(true);
    });
  });

  it('markAllRead marks all notifications as read', () => {
    createRoot(() => {
      notificationStore.clearAll();
      notificationStore.addNotification({ type: 'info', title: 'N1' });
      notificationStore.addNotification({ type: 'error', title: 'N2' });
      expect(notificationStore.unreadCount()).toBe(2);
      notificationStore.markAllRead();
      expect(notificationStore.unreadCount()).toBe(0);
    });
  });

  it('clearAll removes all notifications', () => {
    createRoot(() => {
      notificationStore.addNotification({ type: 'info', title: 'N1' });
      notificationStore.clearAll();
      expect(notificationStore.notifications()).toEqual([]);
    });
  });

  it('removeNotification removes a specific notification', () => {
    createRoot(() => {
      notificationStore.clearAll();
      notificationStore.addNotification({ type: 'info', title: 'Keep' });
      notificationStore.addNotification({ type: 'error', title: 'Remove' });
      const toRemove = notificationStore.notifications().find(n => n.title === 'Remove');
      expect(toRemove).toBeDefined();
      notificationStore.removeNotification(toRemove!.id);
      expect(notificationStore.notifications().find(n => n.title === 'Remove')).toBeUndefined();
      expect(notificationStore.notifications().find(n => n.title === 'Keep')).toBeDefined();
    });
  });

  it('keeps only last 100 notifications', () => {
    createRoot(() => {
      notificationStore.clearAll();
      for (let i = 0; i < 105; i++) {
        notificationStore.addNotification({ type: 'info', title: `N${i}` });
      }
      expect(notificationStore.notifications().length).toBeLessThanOrEqual(100);
    });
  });

  it('addNotification supports optional link', () => {
    createRoot(() => {
      notificationStore.clearAll();
      notificationStore.addNotification({
        type: 'info',
        title: 'Linked',
        link: '/projects/1',
      });
      const n = notificationStore.notifications()[0];
      expect(n.link).toBe('/projects/1');
    });
  });
});
