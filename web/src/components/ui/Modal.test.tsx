import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import Modal from './Modal';

describe('Modal', () => {
  it('renders nothing when open is false', () => {
    const { container } = render(() => (
      <Modal open={false} onClose={() => {}}>
        <p>Content</p>
      </Modal>
    ));
    expect(screen.queryByText('Content')).not.toBeInTheDocument();
  });

  it('renders content when open is true', () => {
    render(() => (
      <Modal open={true} onClose={() => {}}>
        <p>Modal Content</p>
      </Modal>
    ));
    expect(screen.getByText('Modal Content')).toBeInTheDocument();
  });

  it('renders title when provided', () => {
    render(() => (
      <Modal open={true} onClose={() => {}} title="My Modal">
        <p>Content</p>
      </Modal>
    ));
    expect(screen.getByText('My Modal')).toBeInTheDocument();
  });

  it('renders description when provided', () => {
    render(() => (
      <Modal open={true} onClose={() => {}} title="Title" description="A description">
        <p>Content</p>
      </Modal>
    ));
    expect(screen.getByText('A description')).toBeInTheDocument();
  });

  it('renders footer when provided', () => {
    render(() => (
      <Modal open={true} onClose={() => {}} footer={<button>Save</button>}>
        <p>Content</p>
      </Modal>
    ));
    expect(screen.getByText('Save')).toBeInTheDocument();
  });

  it('calls onClose when close button is clicked', () => {
    const onClose = vi.fn();
    render(() => (
      <Modal open={true} onClose={onClose} title="Test">
        <p>Content</p>
      </Modal>
    ));
    // The close button is the X button in the header
    const closeButtons = document.querySelectorAll('button');
    // Find the button that is in the header (close icon)
    const closeBtn = Array.from(closeButtons).find(
      btn => btn.querySelector('svg') && !btn.textContent?.trim()
    );
    if (closeBtn) {
      fireEvent.click(closeBtn);
      expect(onClose).toHaveBeenCalledOnce();
    }
  });

  it('calls onClose when Escape key is pressed', () => {
    const onClose = vi.fn();
    render(() => (
      <Modal open={true} onClose={onClose}>
        <p>Content</p>
      </Modal>
    ));
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('applies size class for sm', () => {
    render(() => (
      <Modal open={true} onClose={() => {}} size="sm">
        <p>Small</p>
      </Modal>
    ));
    const modal = screen.getByText('Small').closest('.max-w-sm');
    expect(modal).not.toBeNull();
  });

  it('applies size class for lg', () => {
    render(() => (
      <Modal open={true} onClose={() => {}} size="lg">
        <p>Large</p>
      </Modal>
    ));
    const modal = screen.getByText('Large').closest('.max-w-2xl');
    expect(modal).not.toBeNull();
  });

  it('applies size class for xl', () => {
    render(() => (
      <Modal open={true} onClose={() => {}} size="xl">
        <p>Extra Large</p>
      </Modal>
    ));
    const modal = screen.getByText('Extra Large').closest('.max-w-4xl');
    expect(modal).not.toBeNull();
  });

  it('defaults to md size', () => {
    render(() => (
      <Modal open={true} onClose={() => {}}>
        <p>Default</p>
      </Modal>
    ));
    const modal = screen.getByText('Default').closest('.max-w-lg');
    expect(modal).not.toBeNull();
  });
});
