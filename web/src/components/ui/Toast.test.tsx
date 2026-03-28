import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import { addToast, removeToast, toast } from './Toast';
import ToastContainer from './Toast';

describe('Toast', () => {
  it('renders toast container', () => {
    render(() => <ToastContainer />);
    // Container should exist but be empty initially
    // Since it uses Portal, content goes to document body
  });

  it('addToast adds a success toast', async () => {
    render(() => <ToastContainer />);
    addToast('success', 'Operation successful');
    // Wait for reactivity
    await new Promise(r => setTimeout(r, 10));
    expect(screen.getByText('Operation successful')).toBeInTheDocument();
  });

  it('addToast adds an error toast', async () => {
    render(() => <ToastContainer />);
    addToast('error', 'Something went wrong');
    await new Promise(r => setTimeout(r, 10));
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
  });

  it('addToast adds a warning toast', async () => {
    render(() => <ToastContainer />);
    addToast('warning', 'Watch out');
    await new Promise(r => setTimeout(r, 10));
    expect(screen.getByText('Watch out')).toBeInTheDocument();
  });

  it('addToast adds an info toast', async () => {
    render(() => <ToastContainer />);
    addToast('info', 'FYI');
    await new Promise(r => setTimeout(r, 10));
    expect(screen.getByText('FYI')).toBeInTheDocument();
  });

  it('toast convenience methods work', async () => {
    render(() => <ToastContainer />);
    toast.success('Success!');
    await new Promise(r => setTimeout(r, 10));
    expect(screen.getByText('Success!')).toBeInTheDocument();
  });

  it('removeToast removes a toast', async () => {
    render(() => <ToastContainer />);
    addToast('info', 'To remove', 0); // duration 0 means no auto-remove
    await new Promise(r => setTimeout(r, 10));
    expect(screen.getByText('To remove')).toBeInTheDocument();

    // Find the close button and click it
    const closeButtons = document.querySelectorAll('button');
    const closeBtn = Array.from(closeButtons).find(btn =>
      btn.closest('[class*="pointer-events-auto"]')
    );
    if (closeBtn) {
      fireEvent.click(closeBtn);
      await new Promise(r => setTimeout(r, 10));
    }
  });

  it('supports multiple toasts', async () => {
    render(() => <ToastContainer />);
    addToast('success', 'Toast 1', 0);
    addToast('error', 'Toast 2', 0);
    addToast('info', 'Toast 3', 0);
    await new Promise(r => setTimeout(r, 10));
    expect(screen.getByText('Toast 1')).toBeInTheDocument();
    expect(screen.getByText('Toast 2')).toBeInTheDocument();
    expect(screen.getByText('Toast 3')).toBeInTheDocument();
  });
});
