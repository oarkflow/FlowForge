import { describe, it, expect } from 'vitest';
import { render, screen } from '@solidjs/testing-library';
import Card from './Card';

describe('Card', () => {
  it('renders children', () => {
    render(() => <Card><p>Card content</p></Card>);
    expect(screen.getByText('Card content')).toBeInTheDocument();
  });

  it('renders title', () => {
    render(() => <Card title="My Card"><p>Content</p></Card>);
    expect(screen.getByText('My Card')).toBeInTheDocument();
  });

  it('renders description', () => {
    render(() => <Card description="A card description"><p>Content</p></Card>);
    expect(screen.getByText('A card description')).toBeInTheDocument();
  });

  it('renders actions', () => {
    render(() => (
      <Card actions={<button>Action</button>}>
        <p>Content</p>
      </Card>
    ));
    expect(screen.getByText('Action')).toBeInTheDocument();
  });

  it('does not render header when no title, description, or actions', () => {
    const { container } = render(() => <Card><p>Content only</p></Card>);
    const header = container.querySelector('.border-b');
    expect(header).toBeNull();
  });

  it('renders header when title is provided', () => {
    render(() => <Card title="Title"><p>Content</p></Card>);
    const title = screen.getByText('Title');
    expect(title.closest('.border-b')).not.toBeNull();
  });

  it('applies padding by default', () => {
    const { container } = render(() => <Card><p>Content</p></Card>);
    const content = container.querySelector('.p-5');
    expect(content).not.toBeNull();
  });

  it('removes padding when padding is false', () => {
    const { container } = render(() => <Card padding={false}><p>Content</p></Card>);
    // The content div should not have p-5 class
    const content = screen.getByText('Content').parentElement;
    expect(content?.className).not.toContain('p-5');
  });

  it('applies custom class', () => {
    const { container } = render(() => <Card class="custom-card"><p>Content</p></Card>);
    const card = container.firstElementChild;
    expect(card?.className).toContain('custom-card');
  });

  it('renders title and description together', () => {
    render(() => (
      <Card title="Title" description="Description">
        <p>Content</p>
      </Card>
    ));
    expect(screen.getByText('Title')).toBeInTheDocument();
    expect(screen.getByText('Description')).toBeInTheDocument();
  });
});
