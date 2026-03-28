import type { JSX, Component } from 'solid-js';
import { Show } from 'solid-js';
import Modal from './Modal';
import Button from './Button';

export interface ConfirmDialogProps {
  open: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  title?: string;
  description?: string;
  children?: JSX.Element;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: 'danger' | 'primary';
  loading?: boolean;
}

const ConfirmDialog: Component<ConfirmDialogProps> = (props) => {
  const variant = () => props.variant ?? 'danger';

  return (
    <Modal
      open={props.open}
      onClose={props.onCancel}
      title={props.title ?? 'Confirm'}
      description={props.description}
      size="sm"
      footer={
        <>
          <Button variant="ghost" onClick={props.onCancel} disabled={props.loading}>
            {props.cancelLabel ?? 'Cancel'}
          </Button>
          <Button variant={variant()} onClick={props.onConfirm} loading={props.loading}>
            {props.confirmLabel ?? 'Confirm'}
          </Button>
        </>
      }
    >
      <Show when={props.children} fallback={
        <p class="text-sm text-[var(--color-text-secondary)]">
          Are you sure? This action cannot be undone.
        </p>
      }>
        {props.children}
      </Show>
    </Modal>
  );
};

export default ConfirmDialog;
