import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import RequestsPage from './RequestsPage';

const mockData = {
  items: [
    {
      id: 3,
      key_id: 1,
      key_label: 'Primary OpenAI',
      provider: 'openai',
      model: 'gpt-4o-mini',
      prompt_tokens: 20,
      completion_tokens: 8,
      total_tokens: 28,
      latency_ms: 120,
      status: 'ok',
      created_at: '2026-04-16T10:00:00Z'
    },
    {
      id: 2,
      key_id: 1,
      key_label: 'Primary OpenAI',
      provider: 'openai',
      model: 'gpt-4o-mini',
      prompt_tokens: 10,
      completion_tokens: 5,
      total_tokens: 15,
      latency_ms: 95,
      status: 'error',
      error_message: 'timeout',
      created_at: '2026-04-16T09:00:00Z'
    },
    {
      id: 1,
      key_id: 2,
      key_label: 'Claude',
      provider: 'anthropic',
      model: 'claude-3-5-haiku',
      prompt_tokens: 14,
      completion_tokens: 6,
      total_tokens: 20,
      latency_ms: 140,
      status: 'ok',
      created_at: '2026-04-16T08:00:00Z'
    }
  ],
  summary: {
    request_count: 3,
    error_count: 1,
    prompt_tokens: 44,
    completion_tokens: 19,
    total_tokens: 63
  },
  filters: {
    models: ['claude-3-5-haiku', 'gpt-4o-mini']
  }
};

describe('RequestsPage', () => {
  afterEach(() => {
    cleanup();
  });

  it('renders summary cards, model totals, and request rows', () => {
    render(<RequestsPage data={mockData} />);

    expect(screen.getByRole('heading', { name: /requests/i })).toBeInTheDocument();
    expect(screen.getByText('63')).toBeInTheDocument();
    expect(screen.getAllByText('gpt-4o-mini').length).toBeGreaterThan(0);
    expect(screen.getAllByText('claude-3-5-haiku').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Primary OpenAI').length).toBeGreaterThan(0);
    expect(screen.getByText('timeout')).toBeInTheDocument();
  });

  it('filters the rendered table and model totals client-side when the model filter changes', () => {
    render(<RequestsPage data={mockData} />);

    fireEvent.change(screen.getByRole('combobox', { name: /model filter/i }), { target: { value: 'claude-3-5-haiku' } });

    expect(screen.getByText('Claude')).toBeInTheDocument();
    expect(screen.queryByText('Primary OpenAI')).not.toBeInTheDocument();
    expect(screen.getAllByText('20').length).toBeGreaterThan(0);
  });
});
