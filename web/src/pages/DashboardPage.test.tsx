import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import DashboardPage from './DashboardPage';

const mockData = {
  items: [
    {
      model: 'gpt-4o',
      prompt_tokens: 120,
      completion_tokens: 30,
      total_tokens: 150,
      request_count: 4
    },
    {
      model: 'claude-3-7-sonnet',
      prompt_tokens: 80,
      completion_tokens: 20,
      total_tokens: 100,
      request_count: 2
    }
  ],
  summary: {
    prompt_tokens: 200,
    completion_tokens: 50,
    total_tokens: 250,
    request_count: 6
  }
};

describe('DashboardPage', () => {
  it('renders per-model rows and summary totals', () => {
    render(<DashboardPage data={mockData} />);

    expect(screen.getByRole('heading', { name: '仪表盘' })).toBeInTheDocument();
    expect(screen.getByText('gpt-4o')).toBeInTheDocument();
    expect(screen.getByText('claude-3-7-sonnet')).toBeInTheDocument();
    expect(screen.getByRole('cell', { name: '150' })).toBeInTheDocument();
    expect(screen.getAllByText('Prompt token').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Completion token').length).toBeGreaterThan(0);
    expect(screen.getAllByText('Total token').length).toBeGreaterThan(0);
    expect(screen.getByText('250')).toBeInTheDocument();
  });
});
