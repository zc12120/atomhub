import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import ModelsPage from './ModelsPage';

const mockData = {
  items: [{ model: 'gpt-4o-mini', provider: 'openai', key_count: 2, healthy_keys: 2 }]
};

describe('ModelsPage', () => {
  it('renders chinese models copy', () => {
    render(<ModelsPage data={mockData as never} />);

    expect(screen.getByRole('heading', { name: '模型' })).toBeInTheDocument();
    expect(screen.getByText('可用密钥数')).toBeInTheDocument();
  });
});
